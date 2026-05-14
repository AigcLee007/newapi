package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func imageAsyncBillingRequestID(taskID string) string {
	if taskID == "" {
		return ""
	}
	return fmt.Sprintf("image_async:%s", taskID)
}

func FinalizeImageAsyncSuccessBilling(c *gin.Context, task *model.Task, info *relaycommon.RelayInfo, usage *dto.Usage, logContent []string) error {
	if task.PrivateData.ImageAsync == nil {
		return nil
	}
	if task.PrivateData.ImageAsync.BillingFinalized || task.PrivateData.ImageAsync.BillingRefunded {
		return nil
	}
	billingRequestID := imageAsyncBillingRequestID(task.TaskID)
	if model.HasConsumeLogByRequestID(billingRequestID) {
		task.PrivateData.ImageAsync.BillingFinalized = true
		return task.Update()
	}
	if c != nil && info != nil && usage != nil {
		c.Set(common.RequestIdKey, billingRequestID)
		info.FinalPreConsumedQuota = task.Quota
		info.UserId = task.UserId
		info.TokenId = task.PrivateData.TokenId
		info.UsingGroup = task.Group
		info.ChannelId = task.ChannelId
		PostTextConsumeQuota(c, info, usage, logContent)
	} else if usage != nil && usage.TotalTokens > 0 {
		RecalculateTaskQuotaByTokens(context.Background(), task, usage.TotalTokens)
	} else {
		model.UpdateUserUsedQuotaAndRequestCount(task.UserId, task.Quota)
		if task.ChannelId > 0 {
			model.UpdateChannelUsedQuota(task.ChannelId, task.Quota)
		}
	}
	task.PrivateData.ImageAsync.BillingFinalized = true
	return task.Update()
}

func FinalizeImageAsyncFailureRefund(ctx context.Context, task *model.Task, reason string) error {
	if task.PrivateData.ImageAsync == nil {
		return nil
	}
	if task.PrivateData.ImageAsync.BillingRefunded || task.PrivateData.ImageAsync.BillingFinalized {
		return nil
	}
	RefundTaskQuota(ctx, task, reason)
	task.PrivateData.ImageAsync.BillingRefunded = true
	return task.Update()
}
