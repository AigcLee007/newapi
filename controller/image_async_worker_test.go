package controller

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaydto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	hosttypes "github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageAsyncReplayRestoresTokenAndBillingPreference(t *testing.T) {
	for _, preference := range []string{"wallet_only", "subscription_only"} {
		t.Run(preference, func(t *testing.T) {
			task := setupGenericTaskTest(t)
			require.NoError(t, model.DB.AutoMigrate(&model.Token{}))
			t.Cleanup(func() { require.NoError(t, model.DB.Migrator().DropTable(&model.Token{})) })
			var user model.User
			require.NoError(t, model.DB.First(&user, task.UserId).Error)
			user.SetSetting(relaydto.UserSetting{BillingPreference: preference})
			require.NoError(t, model.DB.Model(&user).Update("setting", user.Setting).Error)
			token := model.Token{UserId: user.Id, Key: "async-replay-token", Name: "async-replay", Group: "token-group", UnlimitedQuota: true, RemainQuota: 0}
			require.NoError(t, model.DB.Create(&token).Error)
			task.Group = "vip"
			task.PrivateData = model.TaskPrivateData{TokenId: token.Id, ImageAsync: &model.ImageAsyncPrivateData{UserGroup: "default", PluginKey: "image-bridge"}}
			c, _, err := newImageAsyncReplayContext(task, []byte(`{"model":"test","prompt":"image"}`))
			require.NoError(t, err)
			defer common.CleanupBodyStorage(c)
			info, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
			require.NoError(t, err)
			require.Equal(t, token.Key, info.TokenKey)
			require.True(t, info.TokenUnlimited)
			require.Equal(t, token.Id, info.TokenId)
			require.Equal(t, token.Group, info.TokenGroup)
			require.Equal(t, preference, info.UserSetting.BillingPreference)
			require.Equal(t, "vip", info.UsingGroup)
			require.Equal(t, "default", info.UserGroup)
			require.Equal(t, token.Name, c.GetString("token_name"))
			require.Equal(t, user.Id, common.GetContextKeyInt(c, constant.ContextKeyUserId))
		})
	}
}

func TestApplyImageAsyncBillingSnapshotRestoresFrozenInputs(t *testing.T) {
	price := hosttypes.PriceData{ModelRatio: 2, CompletionRatio: 3, CacheRatio: 0.25, ImageRatio: 4, QuotaToPreConsume: 20_000, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1.5}}
	priceJSON, err := common.Marshal(price)
	require.NoError(t, err)
	input := billingexpr.RequestInput{Body: []byte(`{"n":2,"quality":"high"}`), Headers: map[string]string{"x-quality": "high"}, ImageCount: common.GetPointer(2)}
	inputJSON, err := common.Marshal(input)
	require.NoError(t, err)
	task := &model.Task{Quota: 20_000, Group: "vip", PrivateData: model.TaskPrivateData{
		BillingContext: &model.TaskBillingContext{OtherRatios: map[string]float64{"quality": 2}},
		ImageAsync:     &model.ImageAsyncPrivateData{UserGroup: "vip", PriceDataJSON: priceJSON, ImageQuotaBeforeGroup: 10_000, BillingRequestInputJSON: inputJSON, TieredSnapshot: &billingexpr.BillingSnapshot{ExprString: `tier("image", fixed(0.02) * image_count)`, EstimatedImageCount: common.GetPointer(2)}},
	}}
	info := &relaycommon.RelayInfo{}
	require.NoError(t, applyImageAsyncBillingSnapshot(task, info))
	require.Equal(t, price.CompletionRatio, info.PriceData.CompletionRatio)
	require.Equal(t, price.CacheRatio, info.PriceData.CacheRatio)
	require.Equal(t, price.ImageRatio, info.PriceData.ImageRatio)
	require.Equal(t, price.GroupRatioInfo, info.PriceData.GroupRatioInfo)
	require.Equal(t, map[string]float64{"quality": 2}, info.PriceData.OtherRatios())
	require.Equal(t, float64(10_000), info.ImageQuotaBeforeGroup)
	require.Equal(t, input, *info.BillingRequestInput)
	require.Equal(t, task.PrivateData.ImageAsync.TieredSnapshot, info.TieredBillingSnapshot)
}

func TestImageAsyncPluginSubmissionResumesDurableChildWithoutResubmitting(t *testing.T) {
	c, _ := newImageProtocolTestContext("")
	pinned := imageProtocolTestEndpoint(t)
	c.Set(pluginruntime.ContextKeyPinnedEndpoint, pinned)
	parent := &model.Task{UserId: 71, PrivateData: model.TaskPrivateData{ImageAsync: &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginSubmitted: true, PluginTaskID: "task_child"}}}
	child := imageProtocolTestOutcome(&relaycommon.RelayInfo{UserId: 71}, model.TaskStatusSuccess, "https://provider.example/image.png").Task
	child.TaskID = "task_child"
	deps := pluginProtocolBridgeDeps{
		getByTaskId: func(userID int, taskID string) (*model.Task, bool, error) {
			require.Equal(t, 71, userID)
			require.Equal(t, "task_child", taskID)
			return child, true, nil
		},
		submit: func(*gin.Context, *relaycommon.RelayInfo) (*taskSubmissionOutcome, *dto.TaskError) {
			t.Fatal("persisted child must not be submitted twice")
			return nil, nil
		},
	}
	loaded, err := submitOrResumeImageAsyncPluginTask(c, parent, pinned, deps, nil)
	require.NoError(t, err)
	require.Same(t, child, loaded)
}

func TestImageAsyncPluginSubmissionPersistsBarrierAndStableChildID(t *testing.T) {
	c, _ := newImageProtocolTestContext("")
	pinned := imageProtocolTestEndpoint(t)
	c.Set(pluginruntime.ContextKeyPinnedEndpoint, pinned)
	parent := &model.Task{TaskID: "task_parent", UserId: 71, PrivateData: model.TaskPrivateData{ImageAsync: &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_child"}}}
	persisted := false
	deps := pluginProtocolBridgeDeps{
		getByTaskId: func(int, string) (*model.Task, bool, error) { return nil, false, nil },
		submit: func(_ *gin.Context, info *relaycommon.RelayInfo) (*taskSubmissionOutcome, *dto.TaskError) {
			require.True(t, persisted)
			require.Equal(t, "task_child", info.PublicTaskID)
			require.Equal(t, "image_async:task_child", info.RequestId)
			outcome := imageProtocolTestOutcome(info, model.TaskStatusSubmitted)
			outcome.Task.TaskID = info.PublicTaskID
			return outcome, nil
		},
	}
	child, err := submitOrResumeImageAsyncPluginTask(c, parent, pinned, deps, func() error { persisted = parent.PrivateData.ImageAsync.PluginSubmitted; return nil })
	require.NoError(t, err)
	require.Equal(t, "task_child", child.TaskID)
}

func TestImageAsyncPluginSubmissionNeverRetriesAmbiguousSubmission(t *testing.T) {
	c, _ := newImageProtocolTestContext("")
	pinned := imageProtocolTestEndpoint(t)
	parent := &model.Task{PrivateData: model.TaskPrivateData{ImageAsync: &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginSubmitted: true, PluginTaskID: "task_child"}}}
	deps := pluginProtocolBridgeDeps{getByTaskId: func(int, string) (*model.Task, bool, error) { return nil, false, nil }}
	_, err := submitOrResumeImageAsyncPluginTask(c, parent, pinned, deps, nil)
	require.ErrorIs(t, err, errImageAsyncPluginSubmissionPending)
}

func TestImageAsyncPluginSubmissionLostOwnershipDoesNotFailNewAttempt(t *testing.T) {
	c, _ := newImageProtocolTestContext("")
	pinned := imageProtocolTestEndpoint(t)
	parent := &model.Task{PrivateData: model.TaskPrivateData{ImageAsync: &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_child"}}}
	deps := pluginProtocolBridgeDeps{
		getByTaskId: func(int, string) (*model.Task, bool, error) { return nil, false, nil },
		submit: func(*gin.Context, *relaycommon.RelayInfo) (*taskSubmissionOutcome, *dto.TaskError) {
			t.Fatal("a worker that lost the submission barrier cannot submit")
			return nil, nil
		},
	}
	_, err := submitOrResumeImageAsyncPluginTask(c, parent, pinned, deps, func() error { return errImageAsyncPluginSubmissionPending })
	require.ErrorIs(t, err, errImageAsyncPluginSubmissionPending)
	require.False(t, parent.PrivateData.ImageAsync.PluginSubmitted)
}

func TestRecoverImageAsyncPluginSubmissionWithoutChildExpires(t *testing.T) {
	for _, status := range []model.TaskStatus{model.TaskStatusQueued, model.TaskStatusInProgress} {
		t.Run(string(status), func(t *testing.T) {
			task := setupGenericTaskTest(t)
			t.Setenv("IMAGE_ASYNC_TASK_TIMEOUT_SECONDS", "60")
			t.Setenv("IMAGE_ASYNC_STALE_IN_PROGRESS_SECONDS", "10")
			task.Platform = constant.TaskPlatformSyncTask
			task.Action = constant.ImageAsyncAction
			task.Status = status
			task.Quota = 123
			task.SubmitTime = time.Now().Unix() - 120
			task.StartTime = task.SubmitTime
			task.PrivateData.ImageAsync = &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_missing_child", PluginSubmitted: true, Attempt: 1, WorkerNode: "old-worker", BillingFinalized: true}
			require.NoError(t, model.DB.Save(task).Error)
			recoverImageAsyncTasks(context.Background())
			var stored model.Task
			require.NoError(t, model.DB.First(&stored, task.ID).Error)
			require.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
			require.Equal(t, "100%", stored.Progress)
			require.NotZero(t, stored.FinishTime)
			require.Contains(t, stored.FailReason, "submission")
			require.True(t, stored.PrivateData.ImageAsync.PluginSubmitted, "the ambiguous submission barrier must remain durable")
			require.Equal(t, 123, stored.Quota, "only the child may own billing")
			var user model.User
			require.NoError(t, model.DB.First(&user, task.UserId).Error)
			require.Zero(t, user.Quota, "expiring a parent must never refund the child reservation")
		})
	}
}

func TestRecoverImageAsyncPluginExistingChildStillResumes(t *testing.T) {
	task := setupGenericTaskTest(t)
	t.Setenv("IMAGE_ASYNC_TASK_TIMEOUT_SECONDS", "60")
	t.Setenv("IMAGE_ASYNC_STALE_IN_PROGRESS_SECONDS", "10")
	task.Platform = constant.TaskPlatformSyncTask
	task.Action = constant.ImageAsyncAction
	task.Status = model.TaskStatusInProgress
	task.SubmitTime = time.Now().Unix() - 120
	task.StartTime = task.SubmitTime
	task.PrivateData.ImageAsync = &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_saved_child", PluginSubmitted: true, Attempt: 1, WorkerNode: "old-worker", BillingFinalized: true}
	require.NoError(t, model.DB.Save(task).Error)
	child := &model.Task{TaskID: "task_saved_child", UserId: task.UserId, Platform: "image-bridge", Status: model.TaskStatusInProgress, Quota: 123}
	require.NoError(t, model.DB.Create(child).Error)
	recoverImageAsyncTasks(context.Background())
	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	require.Equal(t, model.TaskStatus(model.TaskStatusQueued), stored.Status)
	require.True(t, stored.PrivateData.ImageAsync.PluginSubmitted)
	var storedChild model.Task
	require.NoError(t, model.DB.First(&storedChild, child.ID).Error)
	require.Equal(t, model.TaskStatus(model.TaskStatusInProgress), storedChild.Status)
	require.Equal(t, 123, storedChild.Quota)
}

func TestImageAsyncPluginWorkerOwnershipDatabase(t *testing.T) {
	db, dialect := openTaskDialectDatabase(t, &model.Task{})
	originalDB := model.DB
	originalMain, originalLog := common.MainDatabaseType(), common.LogDatabaseType()
	model.DB = db
	common.SetDatabaseTypes(dialect, dialect)
	t.Cleanup(func() {
		model.DB = originalDB
		common.SetDatabaseTypes(originalMain, originalLog)
	})
	task := &model.Task{TaskID: "plugin_ownership", UserId: 7, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: model.TaskStatusInProgress,
		PrivateData: model.TaskPrivateData{ImageAsync: &model.ImageAsyncPrivateData{PluginKey: "image-bridge", PluginTaskID: "task_child", Attempt: 1, WorkerNode: "worker"}}}
	require.NoError(t, db.Create(task).Error)
	var recovery, staleWorker model.Task
	require.NoError(t, db.First(&recovery, task.ID).Error)
	require.NoError(t, db.First(&staleWorker, task.ID).Error)
	won, err := task.ClaimImageAsyncPluginSubmission()
	require.NoError(t, err)
	require.True(t, won)
	won, err = staleWorker.ClaimImageAsyncPluginSubmission()
	require.NoError(t, err)
	require.False(t, won)
	won, err = recovery.RequeueImageAsyncPluginTask()
	require.NoError(t, err)
	require.True(t, won)
	require.True(t, recovery.PrivateData.ImageAsync.PluginSubmitted)
	recovery.Status = model.TaskStatusInProgress
	recovery.PrivateData.ImageAsync.Attempt++
	require.NoError(t, db.Save(&recovery).Error)
	won, err = staleWorker.RequeueImageAsyncPluginTask()
	require.NoError(t, err)
	require.False(t, won)
	won, err = task.ExpireImageAsyncPluginTaskWithoutChild(10, 20, "plugin submission expired")
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, model.TaskStatus(model.TaskStatusFailure), task.Status)
	require.True(t, task.PrivateData.ImageAsync.BillingFinalized)
}

func TestImageAsyncFailurePreservesReservationDatabase(t *testing.T) {
	db, dialect := openTaskDialectDatabase(t, &model.Task{}, &model.User{}, &model.Token{}, &model.Log{})
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMain, originalLog := common.MainDatabaseType(), common.LogDatabaseType()
	originalRedis, originalBatch, originalMemory := common.RedisEnabled, common.BatchUpdateEnabled, common.MemoryCacheEnabled
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(dialect, dialect)
	common.RedisEnabled, common.BatchUpdateEnabled, common.MemoryCacheEnabled = false, false, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMain, originalLog)
		common.RedisEnabled, common.BatchUpdateEnabled, common.MemoryCacheEnabled = originalRedis, originalBatch, originalMemory
	})
	for _, scenario := range []string{"stale_quota", "old_attempt", "other_worker"} {
		t.Run(scenario, func(t *testing.T) {
			user := model.User{Username: scenario, AffCode: scenario, Quota: 9000}
			require.NoError(t, db.Create(&user).Error)
			token := model.Token{UserId: user.Id, Key: "failure-" + scenario, RemainQuota: 9000, UsedQuota: 1000}
			require.NoError(t, db.Create(&token).Error)
			task := &model.Task{TaskID: "failure_" + scenario, UserId: user.Id, Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: model.TaskStatusInProgress, Quota: 1000,
				PrivateData: model.TaskPrivateData{TokenId: token.Id, ImageAsync: &model.ImageAsyncPrivateData{Attempt: 1, WorkerNode: "worker"}}}
			require.NoError(t, db.Create(task).Error)
			var timeoutSnapshot model.Task
			require.NoError(t, db.First(&timeoutSnapshot, task.ID).Error)
			won, err := model.ReserveImageAsyncReservation(task, 3000)
			require.NoError(t, err)
			require.True(t, won)
			switch scenario {
			case "old_attempt":
				task.PrivateData.ImageAsync.Attempt++
			case "other_worker":
				task.PrivateData.ImageAsync.WorkerNode = "new-worker"
			}
			require.NoError(t, db.Save(task).Error)
			markImageAsyncFailure(context.Background(), &timeoutSnapshot, model.TaskStatusInProgress, "timeout")
			var stored model.Task
			require.NoError(t, db.First(&stored, task.ID).Error)
			var storedUser model.User
			require.NoError(t, db.First(&storedUser, user.Id).Error)
			var storedToken model.Token
			require.NoError(t, db.First(&storedToken, token.Id).Error)
			if scenario == "stale_quota" {
				require.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
				require.True(t, stored.PrivateData.ImageAsync.BillingRefunded)
				require.Zero(t, stored.Quota)
				require.Equal(t, 10000, storedUser.Quota, "the full 3000 reservation must be refunded")
				require.Equal(t, 10000, storedToken.RemainQuota)
				require.Zero(t, storedToken.UsedQuota)
				require.True(t, timeoutSnapshot.PrivateData.ImageAsync.BillingRefunded)
			} else {
				require.Equal(t, model.TaskStatus(model.TaskStatusInProgress), stored.Status)
				require.Equal(t, 3000, stored.Quota)
				require.False(t, stored.PrivateData.ImageAsync.BillingRefunded)
				require.Equal(t, 7000, storedUser.Quota)
				require.Equal(t, 7000, storedToken.RemainQuota)
			}
		})
	}
}

func TestCaptureImageAsyncBillingSnapshotRoundTripsFrozenInputs(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData:             hosttypes.PriceData{ModelRatio: 2, CompletionRatio: 3, CacheRatio: 0.25, ImageRatio: 4, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1.5}},
		ImageQuotaBeforeGroup: 123,
		BillingRequestInput:   &billingexpr.RequestInput{Body: []byte(`{"n":2,"quality":"high"}`), Headers: map[string]string{"x-quality": "high"}, ImageCount: common.GetPointer(2)},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{ExprString: `tier("image", fixed(0.02) * image_count)`, EstimatedImageCount: common.GetPointer(2)},
	}
	info.PriceData.AddOtherRatio("quality", 2)
	private := &model.ImageAsyncPrivateData{UserGroup: "vip"}
	require.NoError(t, captureImageAsyncBillingSnapshot(info, private))
	stored, err := common.Marshal(private)
	require.NoError(t, err)
	var restored model.ImageAsyncPrivateData
	require.NoError(t, common.Unmarshal(stored, &restored))
	task := &model.Task{PrivateData: model.TaskPrivateData{ImageAsync: &restored, BillingContext: &model.TaskBillingContext{OtherRatios: info.PriceData.OtherRatios()}}}
	replay := &relaycommon.RelayInfo{}
	require.NoError(t, applyImageAsyncBillingSnapshot(task, replay))
	require.Equal(t, info.PriceData, replay.PriceData)
	require.Equal(t, info.BillingRequestInput, replay.BillingRequestInput)
	require.Equal(t, info.TieredBillingSnapshot, replay.TieredBillingSnapshot)
	require.Equal(t, info.ImageQuotaBeforeGroup, replay.ImageQuotaBeforeGroup)
}

func resetImageAsyncWorkerSlotsForTest(t *testing.T) {
	t.Helper()
	imageAsyncWorkerSlotsMu.Lock()
	original := imageAsyncWorkerSlots
	imageAsyncWorkerSlots = nil
	imageAsyncWorkerSlotsMu.Unlock()
	t.Cleanup(func() {
		imageAsyncWorkerSlotsMu.Lock()
		imageAsyncWorkerSlots = original
		imageAsyncWorkerSlotsMu.Unlock()
	})
}

func TestImageAsyncUserGroupConcurrencyDefaults(t *testing.T) {
	require.Equal(t, 2, imageAsyncUserGroupConcurrency(""))
	require.Equal(t, 2, imageAsyncUserGroupConcurrency("default"))
	require.Equal(t, 5, imageAsyncUserGroupConcurrency("vip"))
	require.Equal(t, 10, imageAsyncUserGroupConcurrency("svip"))
	require.Equal(t, 2, imageAsyncUserGroupConcurrency("unknown"))
}

func TestImageAsyncUserGroupConcurrencyOverride(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_USER_GROUP_CONCURRENCY", "default:3,vip=6,svip:12,partner:4")

	require.Equal(t, 3, imageAsyncUserGroupConcurrency("default"))
	require.Equal(t, 6, imageAsyncUserGroupConcurrency("vip"))
	require.Equal(t, 12, imageAsyncUserGroupConcurrency("svip"))
	require.Equal(t, 4, imageAsyncUserGroupConcurrency("partner"))
	require.Equal(t, 3, imageAsyncUserGroupConcurrency("unknown"))
}

func TestImageAsyncWorkerConcurrencyDefaultsToBatchSize(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_WORKER_BATCH_SIZE", "7")

	require.Equal(t, 7, imageAsyncWorkerConcurrency())
}

func TestImageAsyncWorkerConcurrencyOverride(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_WORKER_BATCH_SIZE", "7")
	t.Setenv("IMAGE_ASYNC_WORKER_CONCURRENCY", "4")

	require.Equal(t, 4, imageAsyncWorkerConcurrency())
}

func TestImageAsyncWorkerQueueScanSizeDefault(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_WORKER_BATCH_SIZE", "7")
	t.Setenv("IMAGE_ASYNC_WORKER_CONCURRENCY", "4")

	require.Equal(t, 280, imageAsyncWorkerQueueScanSize())
}

func TestImageAsyncWorkerQueueScanSizeOverride(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_WORKER_QUEUE_SCAN_SIZE", "50")

	require.Equal(t, 50, imageAsyncWorkerQueueScanSize())
}

func TestTryAcquireImageAsyncWorkerSlotRespectsConcurrency(t *testing.T) {
	resetImageAsyncWorkerSlotsForTest(t)
	t.Setenv("IMAGE_ASYNC_WORKER_CONCURRENCY", "2")

	releaseA, ok := tryAcquireImageAsyncWorkerSlot()
	require.True(t, ok)
	releaseB, ok := tryAcquireImageAsyncWorkerSlot()
	require.True(t, ok)
	_, ok = tryAcquireImageAsyncWorkerSlot()
	require.False(t, ok)

	releaseA()
	releaseC, ok := tryAcquireImageAsyncWorkerSlot()
	require.True(t, ok)

	releaseB()
	releaseC()
}

func TestTryAcquireImageAsyncWorkerSlotResizesWhenConcurrencyChanges(t *testing.T) {
	resetImageAsyncWorkerSlotsForTest(t)
	t.Setenv("IMAGE_ASYNC_WORKER_CONCURRENCY", "1")

	releaseA, ok := tryAcquireImageAsyncWorkerSlot()
	require.True(t, ok)
	_, ok = tryAcquireImageAsyncWorkerSlot()
	require.False(t, ok)

	t.Setenv("IMAGE_ASYNC_WORKER_CONCURRENCY", "2")
	releaseB, ok := tryAcquireImageAsyncWorkerSlot()
	require.True(t, ok)
	_, ok = tryAcquireImageAsyncWorkerSlot()
	require.False(t, ok)

	t.Setenv("IMAGE_ASYNC_WORKER_CONCURRENCY", "1")
	_, ok = tryAcquireImageAsyncWorkerSlot()
	require.False(t, ok)

	releaseA()
	_, ok = tryAcquireImageAsyncWorkerSlot()
	require.False(t, ok)

	releaseB()
	releaseC, ok := tryAcquireImageAsyncWorkerSlot()
	require.True(t, ok)
	releaseC()
}
