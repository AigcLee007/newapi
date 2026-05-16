package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type imageAsyncTaskDetail struct {
	TaskID        string                         `json:"task_id"`
	UserID        int                            `json:"user_id"`
	Username      string                         `json:"username,omitempty"`
	Group         string                         `json:"group,omitempty"`
	ChannelID     int                            `json:"channel_id"`
	Model         string                         `json:"model,omitempty"`
	Status        string                         `json:"status"`
	Progress      string                         `json:"progress,omitempty"`
	SubmitTime    int64                          `json:"submit_time"`
	StartTime     int64                          `json:"start_time"`
	FinishTime    int64                          `json:"finish_time"`
	FailReason    string                         `json:"fail_reason,omitempty"`
	Quota         int                            `json:"quota"`
	Data          any                            `json:"data,omitempty"`
	ResultObjects []model.ImageAsyncResultObject `json:"result_objects,omitempty"`
	Request       imageAsyncRequestSummary       `json:"request"`
	Debug         imageAsyncDebugSummary         `json:"debug"`
}

type imageAsyncRequestSummary struct {
	Path        string         `json:"path,omitempty"`
	Method      string         `json:"method,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	BodySize    int64          `json:"body_size,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
}

type imageAsyncDebugSummary struct {
	WorkerNode        string `json:"worker_node,omitempty"`
	Attempt           int    `json:"attempt,omitempty"`
	LastError         string `json:"last_error,omitempty"`
	WebhookConfigured bool   `json:"webhook_configured"`
	WebhookDone       bool   `json:"webhook_done"`
	WebhookAttempts   int    `json:"webhook_attempts,omitempty"`
	BillingFinalized  bool   `json:"billing_finalized"`
	BillingRefunded   bool   `json:"billing_refunded"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
}

func GetImageAsyncTaskDetail(c *gin.Context) {
	taskID := c.Param("task_id")
	userID := c.GetInt("id")
	role := c.GetInt("role")

	var task *model.Task
	var exist bool
	var err error
	if role >= common.RoleAdminUser {
		task, exist, err = model.GetByOnlyTaskId(taskID)
	} else {
		task, exist, err = model.GetByTaskId(userID, taskID)
	}
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if !exist || task.Platform != constant.TaskPlatformSyncTask || task.Action != constant.ImageAsyncAction {
		common.ApiErrorMsg(c, "task_not_exist")
		return
	}

	detail := buildImageAsyncTaskDetail(task)
	common.ApiSuccess(c, detail)
}

func buildImageAsyncTaskDetail(task *model.Task) imageAsyncTaskDetail {
	var username string
	if user, err := model.GetUserCache(task.UserId); err == nil && user != nil {
		username = user.Username
	}

	imageAsync := task.PrivateData.ImageAsync
	detail := imageAsyncTaskDetail{
		TaskID:     task.TaskID,
		UserID:     task.UserId,
		Username:   username,
		Group:      task.Group,
		ChannelID:  task.ChannelId,
		Model:      task.Properties.OriginModelName,
		Status:     string(task.Status),
		Progress:   task.Progress,
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
		FailReason: task.FailReason,
		Quota:      task.Quota,
		Data:       parseImageAsyncTaskData(task.Data),
	}
	if imageAsync == nil {
		return detail
	}

	detail.ResultObjects = imageAsync.ResultObjects
	detail.Request = imageAsyncRequestSummary{
		Path:        imageAsync.RequestPath,
		Method:      imageAsync.Method,
		ContentType: imageAsync.ContentType,
		BodySize:    imageAsync.BodySize,
		Fields:      readImageAsyncRequestFields(imageAsync),
	}
	detail.Debug = imageAsyncDebugSummary{
		WorkerNode:        imageAsync.WorkerNode,
		Attempt:           imageAsync.Attempt,
		LastError:         imageAsync.LastError,
		WebhookConfigured: imageAsync.WebhookURL != "",
		WebhookDone:       imageAsync.WebhookDone,
		WebhookAttempts:   imageAsync.WebhookAttempts,
		BillingFinalized:  imageAsync.BillingFinalized,
		BillingRefunded:   imageAsync.BillingRefunded,
		IdempotencyKey:    imageAsync.IdempotencyKey,
	}
	return detail
}

func parseImageAsyncTaskData(raw []byte) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var parsed any
	if err := common.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	return sanitizeImageAsyncResultData(parsed)
}

func sanitizeImageAsyncResultData(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if key == "b64_json" {
				if text, ok := item.(string); ok && text != "" {
					result[key] = "[base64 omitted]"
					result["b64_json_size"] = len(text)
				}
				continue
			}
			if key == "url" {
				if text, ok := item.(string); ok && strings.HasPrefix(text, "data:image/") {
					result[key] = "[data URL omitted]"
					result["url_size"] = len(text)
					continue
				}
			}
			result[key] = sanitizeImageAsyncResultData(item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, sanitizeImageAsyncResultData(item))
		}
		return result
	default:
		return value
	}
}

func readImageAsyncRequestFields(imageAsync *model.ImageAsyncPrivateData) map[string]any {
	if imageAsync == nil || !strings.Contains(imageAsync.ContentType, "application/json") {
		return nil
	}
	body, err := service.ReadImageAsyncRequestBody(imageAsync.BodyPath, imageAsync.BodySHA256, service.GetImageAsyncMaxBodyBytes())
	if err != nil {
		return nil
	}
	var request map[string]any
	if err := common.Unmarshal(body, &request); err != nil {
		return nil
	}
	return sanitizeImageAsyncRequestFields(request)
}

func sanitizeImageAsyncRequestFields(request map[string]any) map[string]any {
	allow := []string{"model", "prompt", "size", "aspect_ratio", "aspectRatio", "imageSize", "image_size", "quality", "response_format", "n"}
	fields := make(map[string]any)
	for _, key := range allow {
		if value, ok := request[key]; ok {
			fields[key] = value
		}
	}
	if hasImageInput(request) {
		fields["has_image_input"] = true
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func hasImageInput(request map[string]any) bool {
	for _, key := range []string{"image", "images", "mask"} {
		value, ok := request[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return true
			}
		case []any:
			if len(typed) > 0 {
				return true
			}
		default:
			return true
		}
	}
	return false
}
