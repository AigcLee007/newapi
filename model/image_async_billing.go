package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// ReserveImageAsyncReservation commits an increased local reservation together
// with its replay snapshot. A failed checkpoint rolls back funding and token
// deductions, so recovery always sees the full amount that must be refunded.
// False means ownership changed or the funding balance could not cover it.
func ReserveImageAsyncReservation(task *Task, target int) (bool, error) {
	if target < 0 || target > common.MaxQuota {
		return false, errors.New("image async reservation is out of range")
	}
	if err := prepareImageAsyncReservationCache(task, target); err != nil {
		return false, err
	}
	checkpoint := newImageAsyncCacheCheckpoint()
	var stored Task
	reserved := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		image := stored.PrivateData.ImageAsync
		if image == nil || image.PluginKey != "" || image.BillingFinalized || image.BillingRefunded || stored.Status != TaskStatusInProgress {
			return nil
		}
		caller := task.PrivateData.ImageAsync
		if caller == nil || caller.Attempt != image.Attempt || caller.WorkerNode != image.WorkerNode {
			return nil
		}
		if stored.Quota < 0 {
			return errors.New("image async reservation is negative")
		}
		delta := max(target-stored.Quota, 0)
		if delta > 0 {
			switch stored.PrivateData.BillingSource {
			case "", "wallet":
				if err := checkpoint.apply(stored.UserId, getUserCacheKey(stored.UserId), true, delta, true, false); err != nil {
					return err
				}
				result := tx.Model(&User{}).Where("id = ? AND quota >= ?", stored.UserId, delta).Update("quota", gorm.Expr("quota - ?", delta))
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return errors.New("image async wallet quota insufficient")
				}
			case "subscription":
				var sub UserSubscription
				if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", stored.PrivateData.SubscriptionId, stored.UserId).First(&sub).Error; err != nil {
					return err
				}
				if sub.AmountUsed < 0 || sub.AmountUsed > common.MaxWalletQuota-int64(delta) || sub.AmountTotal > 0 && int64(delta) > sub.AmountTotal-sub.AmountUsed {
					return errors.New("image async subscription quota insufficient")
				}
				if err := tx.Model(&sub).Update("amount_used", sub.AmountUsed+int64(delta)).Error; err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported image async funding source %q", stored.PrivateData.BillingSource)
			}
			var token Token
			if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", stored.PrivateData.TokenId, stored.UserId).First(&token).Error; err != nil {
				return err
			}
			if err := checkpoint.apply(token.Id, getTokenCacheKey(token.Key), false, delta, true, token.UnlimitedQuota); err != nil {
				return err
			}
			if !token.UnlimitedQuota && token.RemainQuota < delta || token.UsedQuota > common.MaxWalletQuota-delta {
				return errors.New("image async token quota insufficient")
			}
			if err := tx.Model(&token).Updates(map[string]any{"remain_quota": gorm.Expr("remain_quota - ?", delta), "used_quota": gorm.Expr("used_quota + ?", delta), "accessed_time": common.GetTimestamp()}).Error; err != nil {
				return err
			}
		}
		if snapshot := task.PrivateData.ImageAsync; snapshot != nil {
			image.PriceDataJSON = snapshot.PriceDataJSON
			image.ImageQuotaBeforeGroup = snapshot.ImageQuotaBeforeGroup
			image.TieredSnapshot = snapshot.TieredSnapshot
			image.BillingRequestInputJSON = snapshot.BillingRequestInputJSON
		}
		if task.PrivateData.BillingContext != nil {
			stored.PrivateData.BillingContext = task.PrivateData.BillingContext
		}
		stored.Quota += delta
		if err := tx.Model(&stored).Updates(map[string]any{"quota": stored.Quota, "private_data": stored.PrivateData}).Error; err != nil {
			return err
		}
		reserved = true
		return nil
	})
	checkpoint.finish(err == nil)
	if err != nil || !reserved {
		return false, err
	}
	*task = stored
	return true, nil
}

// RefundImageAsyncReservation atomically releases a local image reservation
// and marks it refunded. Unlike plugin tasks, local image pre-consumption has
// not incremented user/channel usage, so their counters must stay unchanged.
func RefundImageAsyncReservation(task *Task) (bool, error) {
	checkpoint := newImageAsyncCacheCheckpoint()
	var stored Task
	var refunded int
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		image := stored.PrivateData.ImageAsync
		if image == nil || image.PluginKey != "" || image.BillingFinalized || image.BillingRefunded {
			return nil
		}
		quota := stored.Quota
		if quota < 0 {
			return errors.New("image async reservation is negative")
		}
		if quota > 0 {
			switch stored.PrivateData.BillingSource {
			case "", "wallet":
				if err := checkpoint.apply(stored.UserId, getUserCacheKey(stored.UserId), true, quota, false, false); err != nil {
					return err
				}
				result := tx.Model(&User{}).Where("id = ? AND quota <= ?", stored.UserId, common.MaxWalletQuota-quota).Update("quota", gorm.Expr("quota + ?", quota))
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return errors.New("image async refund user is missing or wallet limit exceeded")
				}
			case "subscription":
				var sub UserSubscription
				if stored.PrivateData.SubscriptionId <= 0 {
					return errors.New("image async subscription is missing")
				}
				if err := lockForUpdate(tx).First(&sub, stored.PrivateData.SubscriptionId).Error; err != nil {
					return err
				}
				sub.AmountUsed = max(sub.AmountUsed-int64(quota), 0)
				if err := tx.Save(&sub).Error; err != nil {
					return err
				}
				// Older queued tasks used a random request id; absence of the new
				// stable record must not prevent releasing their saved reservation.
				if err := tx.Model(&SubscriptionPreConsumeRecord{}).Where("request_id = ?", fmt.Sprintf("image_async:%s", stored.TaskID)).Update("status", "refunded").Error; err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported image async funding source %q", stored.PrivateData.BillingSource)
			}
			if stored.PrivateData.TokenId > 0 {
				var token Token
				if err := tx.Where("id = ?", stored.PrivateData.TokenId).Find(&token).Error; err != nil {
					return err
				}
				if token.Id > 0 {
					if err := checkpoint.apply(token.Id, getTokenCacheKey(token.Key), false, quota, false, token.UnlimitedQuota); err != nil {
						return err
					}
				}
				if err := tx.Model(&Token{}).Where("id = ?", stored.PrivateData.TokenId).Updates(map[string]any{"remain_quota": gorm.Expr("remain_quota + ?", quota), "used_quota": gorm.Expr("used_quota - ?", quota)}).Error; err != nil {
					return err
				}
			}
		}
		stored.Quota = 0
		image.BillingRefunded = true
		if err := tx.Model(&stored).Updates(map[string]any{"quota": 0, "private_data": stored.PrivateData}).Error; err != nil {
			return err
		}
		refunded = quota
		return nil
	})
	checkpoint.finish(err == nil)
	if err != nil {
		return false, err
	}
	*task = stored
	return refunded > 0, nil
}
