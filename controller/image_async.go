package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func IsImageAsyncRequest(c *gin.Context) bool {
	if strings.EqualFold(os.Getenv("IMAGE_ASYNC_ENABLED"), "false") {
		return false
	}
	if !strings.EqualFold(c.Query("async"), "true") {
		return false
	}
	switch c.Request.URL.Path {
	case "/v1/images/generations", "/v1/images/edits", "/v1/edits":
		return true
	default:
		return false
	}
}

func RelayImageAsyncSubmit(c *gin.Context) {
	request, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIImage)
	if err != nil {
		imageAsyncError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIImage, request, nil)
	if err != nil {
		imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	imageRequest, _ := request.(*dto.ImageRequest)
	originModelName := relayInfo.OriginModelName
	if originModelName == "" && imageRequest != nil {
		originModelName = imageRequest.Model
	}
	relayInfo.ForcePreConsume = true

	meta := request.GetTokenCountMeta()
	if setting.ShouldCheckPromptSensitive() && meta != nil {
		contains, words := service.CheckSensitiveText(meta.CombineText)
		if contains {
			logger.LogWarn(c, fmt.Sprintf("user sensitive words detected: %s", strings.Join(words, ", ")))
			imageAsyncError(c, http.StatusBadRequest, "sensitive_words_detected", nil)
			return
		}
	}
	tokens, err := service.EstimateRequestToken(c, meta, relayInfo)
	if err != nil {
		imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	relayInfo.SetEstimatePromptTokens(tokens)
	priceData, err := helper.ModelPriceHelper(c, relayInfo, tokens, meta)
	if err != nil {
		imageAsyncError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	bodyStorage, err := common.GetBodyStorage(c)
	if err != nil {
		status := http.StatusBadRequest
		if common.IsRequestBodyTooLargeError(err) || errors.Is(err, common.ErrRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		imageAsyncError(c, status, err.Error(), nil)
		return
	}
	rawBody, err := bodyStorage.Bytes()
	if err != nil {
		imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	bodySum := sha256.Sum256(rawBody)
	bodySHA := hex.EncodeToString(bodySum[:])
	requestHash := imageAsyncRequestHash(relayInfo.UserId, c.Request.Method, c.Request.URL, c.GetHeader("Content-Type"), bodySHA)
	idempotencyKey := imageAsyncIdempotencyKey(c)
	if idempotencyKey != "" {
		item, exist, err := model.GetImageAsyncIdempotency(relayInfo.UserId, idempotencyKey)
		if err != nil {
			imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		if exist {
			if item.RequestHash != requestHash {
				imageAsyncError(c, http.StatusConflict, "idempotency_key_conflict", nil)
				return
			}
			imageAsyncSuccess(c, item.TaskID)
			return
		}
	}

	if !priceData.FreeModel {
		if apiErr := service.PreConsumeBilling(c, priceData.QuotaToPreConsume, relayInfo); apiErr != nil {
			imageAsyncError(c, apiErr.StatusCode, apiErr.Error(), nil)
			return
		}
	}

	taskID := model.GenerateTaskID()
	bodyPath, savedSHA, bodySize, err := service.SaveImageAsyncRequestBody(taskID, rawBody)
	if err != nil {
		if relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
		}
		imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	privateData := model.TaskPrivateData{
		BillingSource:  relayInfo.BillingSource,
		SubscriptionId: relayInfo.SubscriptionId,
		TokenId:        relayInfo.TokenId,
		BillingContext: &model.TaskBillingContext{
			ModelPrice:      relayInfo.PriceData.ModelPrice,
			GroupRatio:      relayInfo.PriceData.GroupRatioInfo.GroupRatio,
			ModelRatio:      relayInfo.PriceData.ModelRatio,
			OtherRatios:     relayInfo.PriceData.OtherRatios,
			OriginModelName: originModelName,
			PerCallBilling:  relayInfo.PriceData.UsePrice,
		},
	}
	replayRelayMode := relayInfo.RelayMode
	if c.Request.URL.Path == "/v1/edits" {
		replayRelayMode = relayconstant.RelayModeImagesEdits
	}
	privateData.ImageAsync = &model.ImageAsyncPrivateData{
		RequestPath:    imageAsyncReplayPath(c.Request.URL),
		Method:         c.Request.Method,
		ContentType:    c.GetHeader("Content-Type"),
		BodyPath:       bodyPath,
		BodySHA256:     savedSHA,
		BodySize:       bodySize,
		RelayMode:      replayRelayMode,
		RelayFormat:    string(types.RelayFormatOpenAIImage),
		WebhookURL:     c.Query("webhook"),
		IdempotencyKey: idempotencyKey,
		RequestHash:    requestHash,
	}
	task := &model.Task{
		TaskID:     taskID,
		UserId:     relayInfo.UserId,
		Group:      relayInfo.UsingGroup,
		SubmitTime: model.GetDBTimestamp(),
		Status:     model.TaskStatusQueued,
		Progress:   "0%",
		ChannelId:  common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		Platform:   constant.TaskPlatformSyncTask,
		Action:     constant.ImageAsyncAction,
		Quota:      priceData.QuotaToPreConsume,
		Data:       json.RawMessage("null"),
		Properties: model.Properties{
			OriginModelName: originModelName,
		},
		PrivateData: privateData,
	}

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		if idempotencyKey != "" {
			return tx.Create(&model.ImageAsyncIdempotency{
				UserId:      relayInfo.UserId,
				Key:         idempotencyKey,
				RequestHash: requestHash,
				TaskID:      taskID,
			}).Error
		}
		return nil
	})
	if err != nil {
		if relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
		}
		_ = service.DeleteImageAsyncRequestBody(bodyPath)
		imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	imageAsyncSuccess(c, taskID)
}

func RelayImageAsyncFetch(c *gin.Context) {
	taskID := c.Param("task_id")
	userID := common.GetContextKeyInt(c, constant.ContextKeyUserId)
	task, exist, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		imageAsyncError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if !exist || task.Platform != constant.TaskPlatformSyncTask || task.Action != constant.ImageAsyncAction {
		imageAsyncError(c, http.StatusNotFound, "task_not_exist", nil)
		return
	}
	imageAsyncSuccess(c, imageAsyncTaskResponse(task))
}

func imageAsyncTaskResponse(task *model.Task) gin.H {
	status := "IN_PROGRESS"
	if task.Status == model.TaskStatusSuccess {
		status = "SUCCESS"
	} else if task.Status == model.TaskStatusFailure {
		status = "FAILURE"
	}
	var data any
	if len(task.Data) > 0 && string(task.Data) != "null" {
		var parsed any
		if err := common.Unmarshal(task.Data, &parsed); err == nil {
			data = parsed
		}
	}
	return gin.H{
		"task_id":     task.TaskID,
		"platform":    string(constant.TaskPlatformSyncTask),
		"action":      constant.ImageAsyncAction,
		"status":      status,
		"fail_reason": task.FailReason,
		"submit_time": task.SubmitTime,
		"start_time":  task.StartTime,
		"finish_time": task.FinishTime,
		"progress":    task.Progress,
		"data":        data,
		"search_item": "",
	}
}

func imageAsyncIdempotencyKey(c *gin.Context) string {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(c.GetHeader("X-Idempotency-Key"))
	}
	return key
}

func imageAsyncRequestHash(userID int, method string, u *url.URL, contentType string, bodySHA string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\n%s\n%s\n%s\n%s", userID, method, imageAsyncHashPath(u), contentType, bodySHA)))
	return hex.EncodeToString(sum[:])
}

func imageAsyncReplayPath(u *url.URL) string {
	q := u.Query()
	q.Del("async")
	q.Del("webhook")
	path := u.Path
	if path == "/v1/edits" {
		path = "/v1/images/edits"
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path
}

func imageAsyncHashPath(u *url.URL) string {
	q := u.Query()
	q.Del("async")
	path := u.Path
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path
}

func imageAsyncSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": "success", "message": "", "data": data})
}

func imageAsyncError(c *gin.Context, status int, message string, data any) {
	c.JSON(status, gin.H{"code": "error", "message": message, "data": data})
}
