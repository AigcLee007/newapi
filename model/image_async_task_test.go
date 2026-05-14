package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQueuedImageAsyncTasksFiltersPlatformActionAndStatus(t *testing.T) {
	truncateTables(t)

	insertTask(t, &Task{TaskID: "queued", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusQueued, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "submitted", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusSubmitted, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "success", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusSuccess, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "other_action", Platform: constant.TaskPlatformSyncTask, Action: "other", Status: TaskStatusQueued, Data: json.RawMessage(`{}`)})

	tasks := GetQueuedImageAsyncTasks(10)
	require.Len(t, tasks, 2)
	assert.Equal(t, "queued", tasks[0].TaskID)
	assert.Equal(t, "submitted", tasks[1].TaskID)
}

func TestGetImageAsyncTerminalTasksNeedBillingFiltersByFlags(t *testing.T) {
	truncateTables(t)

	insertTask(t, &Task{
		TaskID:   "success_needs_billing",
		Platform: constant.TaskPlatformSyncTask,
		Action:   constant.ImageAsyncAction,
		Status:   TaskStatusSuccess,
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{
			BillingFinalized: false,
		}},
		Data: json.RawMessage(`{}`),
	})
	insertTask(t, &Task{
		TaskID:   "success_done",
		Platform: constant.TaskPlatformSyncTask,
		Action:   constant.ImageAsyncAction,
		Status:   TaskStatusSuccess,
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{
			BillingFinalized: true,
		}},
		Data: json.RawMessage(`{}`),
	})
	insertTask(t, &Task{
		TaskID:   "failure_needs_refund",
		Platform: constant.TaskPlatformSyncTask,
		Action:   constant.ImageAsyncAction,
		Status:   TaskStatusFailure,
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{
			BillingRefunded: false,
		}},
		Data: json.RawMessage(`{}`),
	})

	tasks := GetImageAsyncTerminalTasksNeedBilling(10)
	require.Len(t, tasks, 2)
	assert.Equal(t, "success_needs_billing", tasks[0].TaskID)
	assert.Equal(t, "failure_needs_refund", tasks[1].TaskID)
}

func TestImageAsyncIdempotencyUniquePerUserAndKey(t *testing.T) {
	truncateTables(t)

	first := &ImageAsyncIdempotency{UserId: 1, Key: "same-key", RequestHash: "hash-a", TaskID: "task_a"}
	require.NoError(t, DB.Create(first).Error)

	item, exists, err := GetImageAsyncIdempotency(1, "same-key")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, "task_a", item.TaskID)

	duplicate := &ImageAsyncIdempotency{UserId: 1, Key: "same-key", RequestHash: "hash-b", TaskID: "task_b"}
	require.Error(t, DB.Create(duplicate).Error)

	otherUser := &ImageAsyncIdempotency{UserId: 2, Key: "same-key", RequestHash: "hash-b", TaskID: "task_b"}
	require.NoError(t, DB.Create(otherUser).Error)
}

func TestHasConsumeLogByRequestID(t *testing.T) {
	truncateTables(t)

	assert.False(t, HasConsumeLogByRequestID("image_async:task_missing"))
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    1,
		CreatedAt: time.Now().Unix(),
		Type:      LogTypeConsume,
		RequestId: "image_async:task_logged",
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    1,
		CreatedAt: time.Now().Unix(),
		Type:      LogTypeRefund,
		RequestId: "image_async:task_refund",
	}).Error)

	assert.True(t, HasConsumeLogByRequestID("image_async:task_logged"))
	assert.False(t, HasConsumeLogByRequestID("image_async:task_refund"))
}
