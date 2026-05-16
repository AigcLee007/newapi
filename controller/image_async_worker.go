package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

func StartImageAsyncWorker() {
	if !imageAsyncFeatureEnabled() || !imageAsyncWorkerEnabled() {
		return
	}
	if !common.IsMasterNode {
		return
	}
	gopool.Go(func() {
		ticker := time.NewTicker(imageAsyncWorkerInterval())
		defer ticker.Stop()
		for {
			runImageAsyncWorkerOnce()
			<-ticker.C
		}
	})
}

func runImageAsyncWorkerOnce() {
	ctx := context.Background()
	recoverImageAsyncTasks(ctx)
	processQueuedImageAsyncTasks(ctx)
	compensateImageAsyncBilling(ctx)
	sendPendingImageAsyncWebhooks(ctx)
	cleanupImageAsyncBodies()
}

func processQueuedImageAsyncTasks(ctx context.Context) {
	tasks := model.GetQueuedImageAsyncTasks(imageAsyncWorkerQueueScanSize())
	if len(tasks) == 0 {
		return
	}

	concurrency := imageAsyncWorkerConcurrency()
	if concurrency <= 1 {
		for _, task := range tasks {
			if canStartImageAsyncTask(task, nil) {
				processImageAsyncTask(ctx, task)
			}
		}
		return
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	plannedByUser := make(map[int]int)

	for _, task := range tasks {
		if !canStartImageAsyncTask(task, plannedByUser) {
			continue
		}
		plannedByUser[task.UserId]++
		sem <- struct{}{}
		wg.Add(1)
		go func(task *model.Task) {
			defer wg.Done()
			defer func() { <-sem }()
			processImageAsyncTask(ctx, task)
		}(task)
	}
	wg.Wait()
}

func canStartImageAsyncTask(task *model.Task, plannedByUser map[int]int) bool {
	limit := imageAsyncUserGroupConcurrency(task.Group)
	if limit <= 0 {
		return false
	}
	running := model.CountRunningImageAsyncTasksByUser(task.UserId)
	if plannedByUser != nil {
		running += plannedByUser[task.UserId]
	}
	return running < limit
}

func processImageAsyncTask(ctx context.Context, task *model.Task) {
	if task.PrivateData.ImageAsync == nil {
		markImageAsyncFailure(ctx, task, task.Status, "missing image async private data")
		return
	}
	oldStatus := task.Status
	now := time.Now().Unix()
	task.Status = model.TaskStatusInProgress
	task.StartTime = now
	task.Progress = "10%"
	task.PrivateData.ImageAsync.Attempt++
	task.PrivateData.ImageAsync.WorkerNode, _ = os.Hostname()
	won, err := task.UpdateWithStatus(oldStatus)
	if err != nil || !won {
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("claim image async task %s failed: %s", task.TaskID, err.Error()))
		}
		return
	}

	body, err := service.ReadImageAsyncRequestBody(task.PrivateData.ImageAsync.BodyPath, task.PrivateData.ImageAsync.BodySHA256, service.GetImageAsyncMaxBodyBytes())
	if err != nil {
		markImageAsyncFailure(ctx, task, model.TaskStatusInProgress, sanitizeImageAsyncError(err))
		return
	}

	result, relayErr := runImageAsyncRelayWithRetry(task, body)
	if relayErr != nil {
		markImageAsyncFailure(ctx, task, model.TaskStatusInProgress, sanitizeImageAsyncError(relayErr))
		return
	}
	defer common.CleanupBodyStorage(result.Context)
	if !json.Valid(result.Body) {
		markImageAsyncFailure(ctx, task, model.TaskStatusInProgress, "upstream returned invalid json")
		return
	}
	normalizedBody, err := service.NormalizeImageAsyncResultStorage(result.Context.Request.Context(), task, result.Body)
	if err != nil {
		markImageAsyncFailure(ctx, task, model.TaskStatusInProgress, sanitizeImageAsyncError(err))
		return
	}

	task.Status = model.TaskStatusSuccess
	task.Progress = "100%"
	task.FinishTime = time.Now().Unix()
	task.FailReason = ""
	task.Data = json.RawMessage(normalizedBody)
	task.ChannelId = result.ChannelID
	won, err = task.UpdateWithStatus(model.TaskStatusInProgress)
	if err != nil || !won {
		service.CleanupImageAsyncResultObjects(ctx, task)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("complete image async task %s failed: %s", task.TaskID, err.Error()))
		}
		return
	}
	_ = service.FinalizeImageAsyncSuccessBilling(result.Context, task, result.RelayInfo, result.Usage, result.LogContent)
	_ = sendImageAsyncWebhookForTask(ctx, task)
}

type imageAsyncRelayResult struct {
	Body       []byte
	Usage      *dto.Usage
	ChannelID  int
	Context    *gin.Context
	RelayInfo  *relaycommon.RelayInfo
	LogContent []string
}

func runImageAsyncRelayWithRetry(task *model.Task, body []byte) (*imageAsyncRelayResult, error) {
	var lastErr error
	var lastChannelID int
	for retry := 0; retry <= common.RetryTimes; retry++ {
		c, recorder, err := newImageAsyncReplayContext(task, body)
		if err != nil {
			return nil, err
		}
		request, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIImage)
		if err != nil {
			common.CleanupBodyStorage(c)
			return nil, err
		}
		if task.Properties.OriginModelName != "" {
			request.SetModelName(task.Properties.OriginModelName)
			common.SetContextKey(c, constant.ContextKeyOriginalModel, task.Properties.OriginModelName)
			c.Set("original_model", task.Properties.OriginModelName)
		}
		relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIImage, request, nil)
		if err != nil {
			common.CleanupBodyStorage(c)
			return nil, err
		}
		if relayInfo.OriginModelName == "" {
			relayInfo.OriginModelName = task.Properties.OriginModelName
		}
		applyImageAsyncBillingSnapshot(task, relayInfo)
		relayInfo.ChannelMeta = &relaycommon.ChannelMeta{}
		retryParam := &service.RetryParam{
			Ctx:        c,
			TokenGroup: relayInfo.TokenGroup,
			ModelName:  relayInfo.OriginModelName,
			Retry:      common.GetPointer(retry),
		}
		channel, channelErr := getChannel(c, relayInfo, retryParam)
		if channelErr != nil {
			lastErr = channelErr
			common.CleanupBodyStorage(c)
			break
		}
		lastChannelID = channel.Id
		addUsedChannel(c, channel.Id)
		usage, logContent, apiErr := relay.RunImageRelay(c, relayInfo)
		if apiErr == nil && recorder.Code < http.StatusBadRequest && recorder.Body.Len() > 0 {
			return &imageAsyncRelayResult{
				Body:       recorder.Body.Bytes(),
				Usage:      usage,
				ChannelID:  channel.Id,
				Context:    c,
				RelayInfo:  relayInfo,
				LogContent: logContent,
			}, nil
		}
		if apiErr == nil {
			apiErr = types.NewErrorWithStatusCode(fmt.Errorf("image async upstream returned status %d", recorder.Code), types.ErrorCodeBadResponseStatusCode, recorder.Code, types.ErrOptionWithSkipRetry())
		}
		lastErr = apiErr
		processChannelError(c, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()), apiErr)
		if !shouldRetry(c, apiErr, common.RetryTimes-retry) {
			common.CleanupBodyStorage(c)
			break
		}
		common.CleanupBodyStorage(c)
	}
	_ = lastChannelID
	return nil, lastErr
}

func newImageAsyncReplayContext(task *model.Task, body []byte) (*gin.Context, *httptest.ResponseRecorder, error) {
	imageAsync := task.PrivateData.ImageAsync
	target := imageAsync.RequestPath
	if target == "" {
		target = "/v1/images/generations"
	}
	u, err := url.Parse(target)
	if err != nil {
		return nil, nil, err
	}
	method := imageAsync.Method
	if method == "" {
		method = http.MethodPost
	}
	req, err := http.NewRequest(method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	if imageAsync.ContentType != "" {
		req.Header.Set("Content-Type", imageAsync.ContentType)
	}
	req.ContentLength = int64(len(body))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		return nil, nil, err
	}
	c.Set(common.KeyBodyStorage, storage)
	common.SetContextKey(c, constant.ContextKeyUserId, task.UserId)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, task.Group)
	common.SetContextKey(c, constant.ContextKeyUserGroup, task.Group)
	common.SetContextKey(c, constant.ContextKeyTokenGroup, task.Group)
	common.SetContextKey(c, constant.ContextKeyTokenId, task.PrivateData.TokenId)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, task.Properties.OriginModelName)
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
	c.Set("original_model", task.Properties.OriginModelName)
	if imageAsync.RelayMode != 0 {
		c.Set("relay_mode", imageAsync.RelayMode)
	}
	return c, recorder, nil
}

func applyImageAsyncBillingSnapshot(task *model.Task, info *relaycommon.RelayInfo) {
	if bc := task.PrivateData.BillingContext; bc != nil {
		info.PriceData.ModelPrice = bc.ModelPrice
		info.PriceData.GroupRatioInfo.GroupRatio = bc.GroupRatio
		info.PriceData.ModelRatio = bc.ModelRatio
		info.PriceData.OtherRatios = bc.OtherRatios
		info.PriceData.UsePrice = bc.PerCallBilling
	}
	info.BillingSource = task.PrivateData.BillingSource
	info.SubscriptionId = task.PrivateData.SubscriptionId
	info.TokenId = task.PrivateData.TokenId
	info.FinalPreConsumedQuota = task.Quota
	info.UsingGroup = task.Group
	info.UserGroup = task.Group
	info.TokenGroup = task.Group
}

func markImageAsyncFailure(ctx context.Context, task *model.Task, from model.TaskStatus, reason string) {
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = time.Now().Unix()
	task.FailReason = reason
	if task.PrivateData.ImageAsync != nil {
		task.PrivateData.ImageAsync.LastError = reason
	}
	won, err := task.UpdateWithStatus(from)
	if err != nil || !won {
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("mark image async task %s failed: %s", task.TaskID, err.Error()))
		}
		return
	}
	_ = service.FinalizeImageAsyncFailureRefund(ctx, task, reason)
	_ = sendImageAsyncWebhookForTask(ctx, task)
}

func recoverImageAsyncTasks(ctx context.Context) {
	now := time.Now().Unix()
	for _, task := range model.GetTimedOutImageAsyncTasks(now-imageAsyncTaskTimeoutSeconds(), imageAsyncWorkerBatchSize()) {
		markImageAsyncFailure(ctx, task, task.Status, "image async task timeout")
	}
	for _, task := range model.GetRecoverableImageAsyncTasks(now-imageAsyncStaleSeconds(), imageAsyncWorkerBatchSize()) {
		markImageAsyncFailure(ctx, task, model.TaskStatusInProgress, "image async task stale; failed to avoid duplicate upstream execution")
	}
}

func compensateImageAsyncBilling(ctx context.Context) {
	for _, task := range model.GetImageAsyncTerminalTasksNeedBilling(imageAsyncWorkerBatchSize()) {
		if task.Status == model.TaskStatusSuccess {
			_ = service.FinalizeImageAsyncSuccessBilling(nil, task, nil, nil, nil)
		} else if task.Status == model.TaskStatusFailure {
			_ = service.FinalizeImageAsyncFailureRefund(ctx, task, task.FailReason)
		}
	}
}

func sendPendingImageAsyncWebhooks(ctx context.Context) {
	for _, task := range model.GetImageAsyncTasksNeedWebhook(time.Now().Unix(), imageAsyncWorkerBatchSize()) {
		_ = sendImageAsyncWebhookForTask(ctx, task)
	}
}

func sendImageAsyncWebhookForTask(ctx context.Context, task *model.Task) error {
	imageAsync := task.PrivateData.ImageAsync
	if imageAsync == nil || imageAsync.WebhookURL == "" || imageAsync.WebhookDone || imageAsync.WebhookAttempts >= imageAsyncWebhookRetryLimit() {
		return nil
	}
	payload := gin.H{"code": "success", "message": "", "data": imageAsyncTaskResponse(task)}
	err := service.SendImageAsyncWebhook(task, payload)
	if err == nil {
		imageAsync.WebhookDone = true
		imageAsync.LastError = ""
		return task.Update()
	}
	imageAsync.WebhookAttempts++
	imageAsync.LastError = sanitizeImageAsyncError(err)
	backoff := []int64{60, 300, 900}
	idx := imageAsync.WebhookAttempts - 1
	if idx >= len(backoff) {
		idx = len(backoff) - 1
	}
	imageAsync.NextWebhookTime = time.Now().Unix() + backoff[idx]
	logger.LogWarn(ctx, fmt.Sprintf("image async webhook task %s failed: %s", task.TaskID, imageAsync.LastError))
	return task.Update()
}

func cleanupImageAsyncBodies() {
	ttlHours := int64(72)
	if raw := os.Getenv("IMAGE_ASYNC_BODY_TTL_HOURS"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			ttlHours = parsed
		}
	}
	cutoff := time.Now().Add(-time.Duration(ttlHours) * time.Hour).Unix()
	for _, task := range model.GetTerminalImageAsyncTasksOlderThan(cutoff, imageAsyncWorkerBatchSize()) {
		if task.PrivateData.ImageAsync == nil || task.PrivateData.ImageAsync.BodyPath == "" {
			continue
		}
		if err := service.DeleteImageAsyncRequestBody(task.PrivateData.ImageAsync.BodyPath); err == nil {
			task.PrivateData.ImageAsync.BodyPath = ""
			_ = task.Update()
		}
	}
}

func sanitizeImageAsyncError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", " ")
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return common.MaskSensitiveInfo(msg)
}

func imageAsyncWorkerInterval() time.Duration {
	if raw := os.Getenv("IMAGE_ASYNC_WORKER_INTERVAL_SECONDS"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			return time.Duration(parsed) * time.Second
		}
	}
	return 2 * time.Second
}

func imageAsyncWorkerConcurrency() int {
	if raw := os.Getenv("IMAGE_ASYNC_WORKER_CONCURRENCY"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	return imageAsyncWorkerBatchSize()
}

func imageAsyncFeatureEnabled() bool {
	return !strings.EqualFold(os.Getenv("IMAGE_ASYNC_ENABLED"), "false")
}

func imageAsyncWorkerEnabled() bool {
	return !strings.EqualFold(os.Getenv("IMAGE_ASYNC_WORKER_ENABLED"), "false")
}

func imageAsyncWorkerBatchSize() int {
	if raw := os.Getenv("IMAGE_ASYNC_WORKER_BATCH_SIZE"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 10
}

func imageAsyncWorkerQueueScanSize() int {
	if raw := os.Getenv("IMAGE_ASYNC_WORKER_QUEUE_SCAN_SIZE"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	batchSize := imageAsyncWorkerBatchSize()
	concurrency := imageAsyncWorkerConcurrency()
	scanSize := batchSize * concurrency * 10
	if scanSize < 100 {
		return 100
	}
	return scanSize
}

func imageAsyncUserGroupConcurrency(group string) int {
	limits := map[string]int{
		"default": 2,
		"vip":     5,
		"svip":    10,
	}
	if raw := strings.TrimSpace(os.Getenv("IMAGE_ASYNC_USER_GROUP_CONCURRENCY")); raw != "" {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';'
		}) {
			key, value, ok := strings.Cut(part, ":")
			if !ok {
				key, value, ok = strings.Cut(part, "=")
			}
			if !ok {
				continue
			}
			key = strings.ToLower(strings.TrimSpace(key))
			if key == "" {
				continue
			}
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil && parsed > 0 {
				limits[key] = parsed
			}
		}
	}
	key := strings.ToLower(strings.TrimSpace(group))
	if key == "" {
		key = "default"
	}
	if limit, ok := limits[key]; ok {
		return limit
	}
	return limits["default"]
}

func imageAsyncTaskTimeoutSeconds() int64 {
	if raw := os.Getenv("IMAGE_ASYNC_TASK_TIMEOUT_SECONDS"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 600
}

func imageAsyncStaleSeconds() int64 {
	if raw := os.Getenv("IMAGE_ASYNC_STALE_IN_PROGRESS_SECONDS"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 900
}

func imageAsyncMaxAttempts() int {
	if raw := os.Getenv("IMAGE_ASYNC_MAX_ATTEMPTS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 3
}

func imageAsyncWebhookRetryLimit() int {
	if raw := os.Getenv("IMAGE_ASYNC_WEBHOOK_RETRY"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 3
}
