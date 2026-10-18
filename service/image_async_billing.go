package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
)

func imageAsyncBillingRequestID(taskID string) string {
	if taskID == "" {
		return ""
	}
	return fmt.Sprintf("image_async:%s", taskID)
}

// RestoreImageAsyncBillingSession reconstructs the billing session for an
// already-reserved image task. The durable task is the owner of the original
// reservation; this function must never call a pre-consume operation.
func RestoreImageAsyncBillingSession(task *model.Task, info *relaycommon.RelayInfo) error {
	if task == nil || task.PrivateData.ImageAsync == nil {
		return errors.New("image async task billing data is missing")
	}
	if info == nil {
		return errors.New("image async relay info is missing")
	}
	if task.Quota < 0 {
		return fmt.Errorf("image async task has negative reserved quota: %d", task.Quota)
	}

	info.UserId = task.UserId
	info.TokenId = task.PrivateData.TokenId
	info.SubscriptionId = task.PrivateData.SubscriptionId
	info.BillingSource = task.PrivateData.BillingSource
	info.RequestId = imageAsyncBillingRequestID(task.TaskID)
	info.FinalPreConsumedQuota = task.Quota
	info.ForcePreConsume = true
	if info.BillingSource == "" {
		info.BillingSource = BillingSourceWallet
	}

	if info.TokenId > 0 {
		token, err := model.GetTokenById(info.TokenId)
		if err != nil {
			return fmt.Errorf("load image async billing token: %w", err)
		}
		info.TokenKey = token.Key
		info.TokenUnlimited = token.UnlimitedQuota
	}

	if task.Quota == 0 {
		info.Billing = nil
		return nil
	}

	var funding FundingSource
	switch info.BillingSource {
	case BillingSourceWallet:
		funding = &WalletFunding{userId: task.UserId, consumed: task.Quota}
	case BillingSourceSubscription:
		if task.PrivateData.SubscriptionId <= 0 {
			return errors.New("image async subscription id is missing")
		}
		funding = &SubscriptionFunding{
			requestId:      info.RequestId,
			userId:         task.UserId,
			modelName:      task.Properties.OriginModelName,
			amount:         int64(task.Quota),
			subscriptionId: task.PrivateData.SubscriptionId,
			preConsumed:    int64(task.Quota),
		}
	default:
		return fmt.Errorf("unsupported image async billing source %q", info.BillingSource)
	}

	info.Billing = &BillingSession{
		relayInfo:        info,
		funding:          funding,
		preConsumedQuota: task.Quota,
		tokenConsumed:    task.Quota,
	}
	info.BillingSource = funding.Source()
	return nil
}

func FinalizeImageAsyncSuccessBilling(c *gin.Context, task *model.Task, info *relaycommon.RelayInfo, usage *dto.Usage, logContent []*common.Message) error {
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
		info.UserGroup = task.GetImageAsyncUserGroup()
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
	quota := task.Quota
	refunded, err := model.RefundImageAsyncReservation(task)
	if err != nil {
		return err
	}
	if refunded {
		other := taskBillingOther(task)
		other.SetPublic("task_id", task.TaskID)
		other.SetPublic("reason", reason)
		model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{UserId: task.UserId, LogType: model.LogTypeRefund, ChannelId: task.ChannelId, ModelName: taskModelName(task), Quota: quota, TokenId: task.PrivateData.TokenId, Group: task.Group, Other: other})
	}
	return nil
}
