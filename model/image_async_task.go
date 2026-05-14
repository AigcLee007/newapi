package model

import "github.com/QuantumNous/new-api/constant"

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
	var tasks []*Task
	err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("status NOT IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
		Where("submit_time < ?", cutoffUnix).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetImageAsyncTerminalTasksNeedBilling(limit int) []*Task {
	var tasks []*Task
	err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
		Order("id asc").
		Limit(limit * 3).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	filtered := make([]*Task, 0, limit)
	for _, task := range tasks {
		if task.PrivateData.ImageAsync == nil {
			continue
		}
		if task.Status == TaskStatusSuccess && !task.PrivateData.ImageAsync.BillingFinalized {
			filtered = append(filtered, task)
		}
		if task.Status == TaskStatusFailure && !task.PrivateData.ImageAsync.BillingRefunded {
			filtered = append(filtered, task)
		}
		if len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func GetImageAsyncTasksNeedWebhook(nowUnix int64, limit int) []*Task {
	var tasks []*Task
	err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
		Order("id asc").
		Limit(limit * 3).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	filtered := make([]*Task, 0, limit)
	for _, task := range tasks {
		imageAsync := task.PrivateData.ImageAsync
		if imageAsync == nil || imageAsync.WebhookURL == "" || imageAsync.WebhookDone {
			continue
		}
		if imageAsync.NextWebhookTime == 0 || imageAsync.NextWebhookTime <= nowUnix {
			filtered = append(filtered, task)
		}
		if len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func GetTerminalImageAsyncTasksOlderThan(cutoffUnix int64, limit int) []*Task {
	var tasks []*Task
	err := DB.Where("platform = ? and action = ?", constant.TaskPlatformSyncTask, constant.ImageAsyncAction).
		Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
		Where("finish_time > 0 and finish_time < ?", cutoffUnix).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}
