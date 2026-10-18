package model

import (
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

// FailImageAsyncTask changes only the terminal state of the current attempt.
// Timeout scans can hold an older quota snapshot than the worker's durable
// reservation checkpoint, so refunding must use the freshly locked row.
func (task *Task) FailImageAsyncTask(from TaskStatus, finishTime int64, reason string) (bool, error) {
	var stored Task
	won := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		if stored.Platform != constant.TaskPlatformSyncTask || stored.Action != constant.ImageAsyncAction ||
			stored.Status != from || stored.Status == TaskStatusSuccess || stored.Status == TaskStatusFailure {
			return nil
		}
		image, caller := stored.PrivateData.ImageAsync, task.PrivateData.ImageAsync
		if (image == nil) != (caller == nil) || image != nil && (image.Attempt != caller.Attempt || image.WorkerNode != caller.WorkerNode) {
			return nil
		}
		stored.Status = TaskStatusFailure
		stored.Progress = "100%"
		stored.FinishTime = finishTime
		stored.FailReason = reason
		updates := map[string]any{"status": stored.Status, "progress": stored.Progress, "finish_time": stored.FinishTime, "fail_reason": stored.FailReason}
		if image != nil {
			image.LastError = reason
			updates["private_data"] = stored.PrivateData
		}
		result := tx.Model(&stored).Where("status = ?", from).Updates(updates)
		won = result.RowsAffected == 1
		return result.Error
	})
	if err != nil || !won {
		return false, err
	}
	*task = stored
	return true, nil
}

// ClaimImageAsyncPluginSubmission persists the one-way submission barrier only
// while this attempt still owns the task. Compare JSON state in Go so the
// ownership guard works with SQLite, MySQL and PostgreSQL alike.
func (task *Task) ClaimImageAsyncPluginSubmission() (bool, error) {
	var stored Task
	won := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		image, caller := stored.PrivateData.ImageAsync, task.PrivateData.ImageAsync
		if stored.Platform != constant.TaskPlatformSyncTask || stored.Action != constant.ImageAsyncAction || stored.Status != TaskStatusInProgress ||
			image == nil || caller == nil || image.PluginKey == "" || image.PluginTaskID == "" || image.PluginSubmitted ||
			image.Attempt != caller.Attempt || image.WorkerNode != caller.WorkerNode || image.PluginKey != caller.PluginKey || image.PluginTaskID != caller.PluginTaskID {
			return nil
		}
		image.PluginSubmitted = true
		result := tx.Model(&stored).Where("status = ?", TaskStatusInProgress).Update("private_data", stored.PrivateData)
		won = result.RowsAffected == 1
		return result.Error
	})
	if err != nil || !won {
		return false, err
	}
	*task = stored
	return true, nil
}

// RequeueImageAsyncPluginTask releases a stale attempt without copying its
// private-data snapshot over a barrier that may have just been persisted.
func (task *Task) RequeueImageAsyncPluginTask() (bool, error) {
	var stored Task
	won := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		image, caller := stored.PrivateData.ImageAsync, task.PrivateData.ImageAsync
		if stored.Platform != constant.TaskPlatformSyncTask || stored.Action != constant.ImageAsyncAction || stored.Status != TaskStatusInProgress ||
			image == nil || caller == nil || image.PluginKey == "" || image.Attempt != caller.Attempt || image.WorkerNode != caller.WorkerNode {
			return nil
		}
		result := tx.Model(&stored).Where("status = ?", TaskStatusInProgress).Update("status", TaskStatusQueued)
		won = result.RowsAffected == 1
		stored.Status = TaskStatusQueued
		return result.Error
	})
	if err != nil || !won {
		return false, err
	}
	*task = stored
	return true, nil
}

// ExpireImageAsyncPluginTaskWithoutChild terminates an abandoned parent while
// leaving settlement and refunds exclusively with any plugin child.
func (task *Task) ExpireImageAsyncPluginTaskWithoutChild(cutoffUnix, nowUnix int64, reason string) (bool, error) {
	var stored Task
	won := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		image := stored.PrivateData.ImageAsync
		if stored.Platform != constant.TaskPlatformSyncTask || stored.Action != constant.ImageAsyncAction ||
			stored.Status == TaskStatusSuccess || stored.Status == TaskStatusFailure || stored.SubmitTime >= cutoffUnix || image == nil || image.PluginKey == "" {
			return nil
		}
		var childCount int64
		if err := tx.Model(&Task{}).Where("user_id = ? AND platform = ? AND task_id = ?", stored.UserId, image.PluginKey, image.PluginTaskID).Count(&childCount).Error; err != nil {
			return err
		}
		if childCount > 0 {
			return nil
		}
		previousStatus := stored.Status
		stored.Status = TaskStatusFailure
		stored.Progress = "100%"
		stored.FinishTime = nowUnix
		stored.FailReason = reason
		image.LastError = reason
		image.BillingFinalized = true
		result := tx.Model(&stored).Where("status = ?", previousStatus).Updates(map[string]any{"status": stored.Status, "progress": stored.Progress, "finish_time": stored.FinishTime, "fail_reason": stored.FailReason, "private_data": stored.PrivateData})
		won = result.RowsAffected == 1
		return result.Error
	})
	if err != nil || !won {
		return false, err
	}
	*task = stored
	return true, nil
}

func GetQueuedImageAsyncTasks(limit int) []*Task {
	var tasks []*Task
	err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("status IN ?", []TaskStatus{TaskStatusQueued, TaskStatusSubmitted, TaskStatusNotStart}).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func CountRunningImageAsyncTasksByUser(userId int) int {
	var count int64
	err := DB.Model(&Task{}).
		Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("user_id = ?", userId).
		Where("status = ?", TaskStatusInProgress).
		Count(&count).Error
	if err != nil {
		return 0
	}
	return int(count)
}

func GetRecoverableImageAsyncTasks(cutoffUnix int64, limit int) []*Task {
	var tasks []*Task
	err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("status = ?", TaskStatusInProgress).
		Where("start_time < ?", cutoffUnix).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetTimedOutImageAsyncTasks(cutoffUnix int64, limit int) []*Task {
	if limit <= 0 {
		return nil
	}
	filtered := make([]*Task, 0, limit)
	batchSize := max(100, limit)
	var cursor int64
	for {
		var tasks []*Task
		err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
			Where("status NOT IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
			Where("submit_time < ? AND id > ?", cutoffUnix, cursor).
			Order("id asc").Limit(batchSize).Find(&tasks).Error
		if err != nil {
			return nil
		}
		for _, task := range tasks {
			cursor = task.ID
			if image := task.PrivateData.ImageAsync; image != nil && image.PluginKey != "" {
				var childCount int64
				if err := DB.Model(&Task{}).Where("user_id = ? AND platform = ? AND task_id = ?", task.UserId, image.PluginKey, image.PluginTaskID).Count(&childCount).Error; err != nil {
					return nil
				}
				if childCount > 0 {
					continue
				}
			}
			filtered = append(filtered, task)
			if len(filtered) == limit {
				return filtered
			}
		}
		if len(tasks) < batchSize {
			return filtered
		}
	}
}

func GetImageAsyncTerminalTasksNeedBilling(limit int) []*Task {
	return getImageAsyncTerminalTasks(limit, 0, func(task *Task) bool {
		state := task.PrivateData.ImageAsync
		return !state.BillingFinalized && !state.BillingRefunded
	})
}

func GetImageAsyncTasksNeedWebhook(nowUnix int64, limit int, maxAttempts ...int) []*Task {
	return getImageAsyncTerminalTasks(limit, 0, func(task *Task) bool {
		state := task.PrivateData.ImageAsync
		if state.WebhookURL == "" || state.WebhookDone {
			return false
		}
		if len(maxAttempts) > 0 && state.WebhookAttempts >= maxAttempts[0] {
			return false
		}
		return state.NextWebhookTime <= nowUnix
	})
}

func GetTerminalImageAsyncTasksOlderThan(cutoffUnix int64, limit int) []*Task {
	return getImageAsyncTerminalTasks(limit, cutoffUnix, func(task *Task) bool {
		return task.PrivateData.ImageAsync.BodyPath != ""
	})
}

// getImageAsyncTerminalTasks pages through settled history until it finds a
// complete maintenance batch. JSON state is filtered in Go for all DB dialects.
func getImageAsyncTerminalTasks(limit int, cutoffUnix int64, include func(*Task) bool) []*Task {
	if limit <= 0 {
		return nil
	}
	filtered := make([]*Task, 0, limit)
	batchSize := max(100, limit)
	var cursor int64
	for {
		query := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
			Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
			Where("id > ?", cursor).Order("id asc").Limit(batchSize)
		if cutoffUnix > 0 {
			query = query.Where("finish_time > 0 and finish_time < ?", cutoffUnix)
		}
		var tasks []*Task
		if err := query.Find(&tasks).Error; err != nil {
			return nil
		}
		for _, task := range tasks {
			cursor = task.ID
			if task.PrivateData.ImageAsync != nil && include(task) {
				filtered = append(filtered, task)
				if len(filtered) == limit {
					return filtered
				}
			}
		}
		if len(tasks) < batchSize {
			return filtered
		}
	}
}
