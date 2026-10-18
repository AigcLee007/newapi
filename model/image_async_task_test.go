package model

import (
	"encoding/json"
	"fmt"
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

func TestClaimImageAsyncPluginSubmissionChecksWorkerOwnership(t *testing.T) {
	for _, scenario := range []string{"current_worker", "old_attempt", "other_worker", "already_submitted", "queued", "different_child"} {
		t.Run(scenario, func(t *testing.T) {
			truncateTables(t)
			task := &Task{TaskID: "claim_" + scenario, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusInProgress,
				PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_child", Attempt: 2, WorkerNode: "worker-b"}}}
			insertTask(t, task)
			var caller Task
			require.NoError(t, DB.First(&caller, task.ID).Error)
			switch scenario {
			case "old_attempt":
				caller.PrivateData.ImageAsync.Attempt--
			case "other_worker":
				caller.PrivateData.ImageAsync.WorkerNode = "worker-a"
			case "already_submitted":
				task.PrivateData.ImageAsync.PluginSubmitted = true
				require.NoError(t, DB.Save(task).Error)
			case "queued":
				task.Status = TaskStatusQueued
				require.NoError(t, DB.Save(task).Error)
			case "different_child":
				caller.PrivateData.ImageAsync.PluginTaskID = "task_other"
			}
			won, err := caller.ClaimImageAsyncPluginSubmission()
			require.NoError(t, err)
			require.Equal(t, scenario == "current_worker", won)
			var stored Task
			require.NoError(t, DB.First(&stored, task.ID).Error)
			require.Equal(t, task.PrivateData.ImageAsync.Attempt, stored.PrivateData.ImageAsync.Attempt)
			require.Equal(t, task.PrivateData.ImageAsync.WorkerNode, stored.PrivateData.ImageAsync.WorkerNode)
			require.Equal(t, scenario == "current_worker" || scenario == "already_submitted", stored.PrivateData.ImageAsync.PluginSubmitted)
			if won {
				won, err = caller.ClaimImageAsyncPluginSubmission()
				require.NoError(t, err)
				require.False(t, won, "the same worker cannot pass the barrier twice")
			}
		})
	}
}

func TestRequeueImageAsyncPluginTaskPreservesNewSubmissionBarrier(t *testing.T) {
	truncateTables(t)
	task := &Task{TaskID: "requeue_barrier", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusInProgress,
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_child", Attempt: 1, WorkerNode: "worker"}}}
	insertTask(t, task)
	var recoverySnapshot Task
	require.NoError(t, DB.First(&recoverySnapshot, task.ID).Error)
	won, err := task.ClaimImageAsyncPluginSubmission()
	require.NoError(t, err)
	require.True(t, won)
	won, err = recoverySnapshot.RequeueImageAsyncPluginTask()
	require.NoError(t, err)
	require.True(t, won)
	var stored Task
	require.NoError(t, DB.First(&stored, task.ID).Error)
	require.Equal(t, TaskStatus(TaskStatusQueued), stored.Status)
	require.True(t, stored.PrivateData.ImageAsync.PluginSubmitted)
	// Reusing the same hostname does not give an older attempt ownership.
	stored.Status = TaskStatusInProgress
	stored.PrivateData.ImageAsync.Attempt++
	require.NoError(t, DB.Save(&stored).Error)
	won, err = recoverySnapshot.RequeueImageAsyncPluginTask()
	require.NoError(t, err)
	require.False(t, won)
}

func TestTimedOutImageAsyncTasksSkipsParentsWithSavedChildren(t *testing.T) {
	truncateTables(t)
	insertTask(t, &Task{TaskID: "waiting_parent", UserId: 7, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusInProgress, SubmitTime: 1,
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "saved_child", PluginSubmitted: true}}})
	insertTask(t, &Task{TaskID: "saved_child", UserId: 7, Platform: "image-bridge", Status: TaskStatusInProgress})
	insertTask(t, &Task{TaskID: "missing_child_parent", UserId: 7, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusQueued, SubmitTime: 1,
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "missing_child", PluginSubmitted: true}}})
	tasks := GetTimedOutImageAsyncTasks(10, 1)
	require.Len(t, tasks, 1)
	require.Equal(t, "missing_child_parent", tasks[0].TaskID, "observing a saved child must not starve expired orphan parents behind it")
}

func TestOfficialTaskTimeoutExcludesLocalImageParents(t *testing.T) {
	truncateTables(t)
	for _, task := range []*Task{
		{TaskID: "local_parent", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, SubmitTime: 1, Status: TaskStatusInProgress, Progress: "10%", PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{}}, Data: json.RawMessage(`{}`)},
		{TaskID: "plugin_child", Platform: "alibaba", Action: "image_generation", SubmitTime: 2, Status: TaskStatusInProgress, Progress: "10%", Data: json.RawMessage(`{}`)},
		{TaskID: "other_sync", Platform: constant.TaskPlatformSyncTask, Action: "video_generation", SubmitTime: 3, Status: TaskStatusInProgress, Progress: "10%", Data: json.RawMessage(`{}`)},
	} {
		insertTask(t, task)
	}
	timedOut := GetTimedOutUnfinishedTasks(10, 1)
	require.Len(t, timedOut, 1, "a local parent must not consume the ordinary timeout batch")
	assert.Equal(t, "plugin_child", timedOut[0].TaskID)
	timedOut = GetTimedOutUnfinishedTasks(10, 10)
	require.Len(t, timedOut, 2)
	assert.Equal(t, "other_sync", timedOut[1].TaskID)
}

func TestCountRunningImageAsyncTasksByUser(t *testing.T) {
	truncateTables(t)

	insertTask(t, &Task{TaskID: "running_a", UserId: 1, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusInProgress, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "running_b", UserId: 1, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusInProgress, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "queued", UserId: 1, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusQueued, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "other_user", UserId: 2, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusInProgress, Data: json.RawMessage(`{}`)})
	insertTask(t, &Task{TaskID: "other_action", UserId: 1, Platform: constant.TaskPlatformSyncTask, Action: "other", Status: TaskStatusInProgress, Data: json.RawMessage(`{}`)})

	assert.Equal(t, 2, CountRunningImageAsyncTasksByUser(1))
	assert.Equal(t, 1, CountRunningImageAsyncTasksByUser(2))
}

func TestGetImageAsyncUserGroupPrefersStoredUserGroup(t *testing.T) {
	task := &Task{
		Group: "default",
		PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{
			UserGroup: "vip",
		}},
	}

	assert.Equal(t, "vip", task.GetImageAsyncUserGroup())
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

func TestImageAsyncMaintenanceFindsPendingTasksAfterCompletedHistory(t *testing.T) {
	for _, operation := range []string{"billing", "webhook", "cleanup"} {
		t.Run(operation, func(t *testing.T) {
			truncateTables(t)
			for i := range 101 {
				insertTask(t, &Task{TaskID: fmt.Sprintf("completed_%d", i), Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusSuccess, FinishTime: 1,
					PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{BillingFinalized: true, WebhookURL: "https://example.com/hook", WebhookDone: true}}, Data: json.RawMessage(`{}`)})
			}
			pending := &Task{TaskID: "pending", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusFailure, FinishTime: 1,
				PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{WebhookURL: "https://example.com/hook", BodyPath: "pending.body"}}, Data: json.RawMessage(`{}`)}
			insertTask(t, pending)
			var tasks []*Task
			switch operation {
			case "billing":
				tasks = GetImageAsyncTerminalTasksNeedBilling(1)
			case "webhook":
				tasks = GetImageAsyncTasksNeedWebhook(10, 1)
			case "cleanup":
				tasks = GetTerminalImageAsyncTasksOlderThan(10, 1)
			}
			require.Len(t, tasks, 1)
			assert.Equal(t, pending.TaskID, tasks[0].TaskID)
		})
	}
}
