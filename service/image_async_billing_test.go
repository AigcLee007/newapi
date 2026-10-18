package service

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupImageAsyncBillingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))

	previousDB := model.DB
	previousRedis, previousBatch := common.RedisEnabled, common.BatchUpdateEnabled
	model.DB = db
	common.RedisEnabled, common.BatchUpdateEnabled = false, false
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled, common.BatchUpdateEnabled = previousRedis, previousBatch
	})
	return db
}

func TestRestoreImageAsyncBillingSessionSettlesOnlyTheOutstandingDelta(t *testing.T) {
	db := setupImageAsyncBillingDB(t)
	user := model.User{Username: "image_async_restore", Quota: 9_000}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "image-async-token", RemainQuota: 9_000}
	require.NoError(t, db.Create(&token).Error)
	task := &model.Task{
		TaskID: taskIDForImageAsyncBillingTest,
		UserId: user.Id,
		Quota:  1_000,
		PrivateData: model.TaskPrivateData{
			BillingSource: BillingSourceWallet,
			TokenId:       token.Id,
			ImageAsync:    &model.ImageAsyncPrivateData{},
		},
	}
	info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id}

	require.NoError(t, RestoreImageAsyncBillingSession(task, info))
	require.Equal(t, 1_000, info.Billing.GetPreConsumedQuota())
	var storedUser model.User
	var storedToken model.Token
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	require.NoError(t, db.First(&storedToken, token.Id).Error)
	require.Equal(t, 9_000, storedUser.Quota, "restoring a task must not reserve its quota again")
	require.Equal(t, 9_000, storedToken.RemainQuota, "restoring a task must not reserve token quota again")

	require.NoError(t, info.Billing.Settle(1_500))
	require.NoError(t, info.Billing.Settle(1_500), "settlement must remain idempotent")
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	require.NoError(t, db.First(&storedToken, token.Id).Error)
	require.Equal(t, 8_500, storedUser.Quota)
	require.Equal(t, 8_500, storedToken.RemainQuota)
}

func TestRestoreImageAsyncBillingSessionRefundsTheExistingReservationOnce(t *testing.T) {
	db := setupImageAsyncBillingDB(t)
	user := model.User{Username: "image_async_refund", Quota: 9_000}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "image-async-refund-token", RemainQuota: 9_000}
	require.NoError(t, db.Create(&token).Error)
	task := &model.Task{
		TaskID: taskIDForImageAsyncBillingTest,
		UserId: user.Id,
		Quota:  1_000,
		PrivateData: model.TaskPrivateData{
			BillingSource: BillingSourceWallet,
			TokenId:       token.Id,
			ImageAsync:    &model.ImageAsyncPrivateData{},
		},
	}
	info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id}
	require.NoError(t, RestoreImageAsyncBillingSession(task, info))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)

	info.Billing.Refund(c)
	info.Billing.Refund(c)
	require.Eventually(t, func() bool {
		var storedUser model.User
		var storedToken model.Token
		return db.First(&storedUser, user.Id).Error == nil &&
			db.First(&storedToken, token.Id).Error == nil &&
			storedUser.Quota == 10_000 && storedToken.RemainQuota == 10_000
	}, time.Second, 10*time.Millisecond)
	require.False(t, info.Billing.NeedsRefund())
}

const taskIDForImageAsyncBillingTest = "image_async_billing_test"

func TestPrepareImageBillingRequiresDurableReservationBeforeUpstream(t *testing.T) {
	db := setupImageAsyncBillingDB(t)
	user := model.User{Username: "image_checkpoint", Quota: 9000}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "image-checkpoint", RemainQuota: 9000}
	require.NoError(t, db.Create(&token).Error)
	task := &model.Task{UserId: user.Id, Quota: 1000, PrivateData: model.TaskPrivateData{TokenId: token.Id, ImageAsync: &model.ImageAsyncPrivateData{}}}
	info := &relaycommon.RelayInfo{Request: &dto.ImageRequest{}, PriceData: hosttypes.PriceData{UsePrice: true, ModelPrice: 1000 / common.QuotaPerUnit, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}}}
	require.NoError(t, RestoreImageAsyncBillingSession(task, info))
	checkpointErr := errors.New("checkpoint unavailable")
	called := false
	info.ImageReservationCheckpoint = func() error { called = true; return checkpointErr }
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	apiErr := PrepareImageBillingForRequest(c, info, 1)
	require.True(t, called)
	require.NotNil(t, apiErr)
	require.ErrorIs(t, apiErr.Err, checkpointErr)
}

func TestImageAsyncReservationCheckpointCommitsQuotaAndFundingTogether(t *testing.T) {
	for _, source := range []string{BillingSourceWallet, BillingSourceSubscription} {
		for _, rejectCheckpoint := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reject=%t", source, rejectCheckpoint), func(t *testing.T) {
				db := setupImageAsyncBillingDB(t)
				require.NoError(t, db.AutoMigrate(&model.Task{}, &model.UserSubscription{}))
				user := model.User{Username: "checkpoint", Quota: 9000}
				require.NoError(t, db.Create(&user).Error)
				token := model.Token{UserId: user.Id, Key: "checkpoint", RemainQuota: 9000, UsedQuota: 1000}
				require.NoError(t, db.Create(&token).Error)
				task := &model.Task{TaskID: "checkpoint", UserId: user.Id, Status: model.TaskStatusInProgress, Quota: 1000, PrivateData: model.TaskPrivateData{BillingSource: source, TokenId: token.Id, ImageAsync: &model.ImageAsyncPrivateData{}}}
				var sub model.UserSubscription
				if source == BillingSourceSubscription {
					sub = model.UserSubscription{UserId: user.Id, AmountTotal: 10000, AmountUsed: 1000}
					require.NoError(t, db.Create(&sub).Error)
					task.PrivateData.SubscriptionId = sub.Id
				}
				require.NoError(t, db.Create(task).Error)
				info := &relaycommon.RelayInfo{Request: &dto.ImageRequest{}, PriceData: hosttypes.PriceData{UsePrice: true, ModelPrice: 1000 / common.QuotaPerUnit, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}}}
				require.NoError(t, RestoreImageAsyncBillingSession(task, info))
				info.ImageReservationCheckpoint = func() error {
					task.PrivateData.ImageAsync.ImageQuotaBeforeGroup = 123
					reserved, err := model.ReserveImageAsyncReservation(task, info.PriceData.QuotaToPreConsume)
					if err != nil {
						return err
					}
					if !reserved {
						return errors.New("reservation rejected")
					}
					return RestoreImageAsyncBillingSession(task, info)
				}
				if rejectCheckpoint {
					require.NoError(t, db.Callback().Update().Before("gorm:update").Register("reject-image-checkpoint", func(tx *gorm.DB) {
						if tx.Statement.Table == "tasks" {
							tx.AddError(errors.New("checkpoint unavailable"))
						}
					}))
				}
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
				apiErr := PrepareImageBillingForRequest(c, info, 2)
				var stored model.Task
				require.NoError(t, db.First(&stored, task.ID).Error)
				require.NoError(t, db.First(&user, user.Id).Error)
				require.NoError(t, db.First(&token, token.Id).Error)
				if rejectCheckpoint {
					require.NotNil(t, apiErr)
					assert.Equal(t, 1000, stored.Quota)
					assert.Equal(t, 9000, user.Quota)
					assert.Equal(t, 9000, token.RemainQuota)
					assert.Equal(t, 1000, token.UsedQuota)
					assert.Zero(t, stored.PrivateData.ImageAsync.ImageQuotaBeforeGroup)
				} else {
					require.Nil(t, apiErr)
					assert.Equal(t, 2000, stored.Quota)
					assert.Equal(t, 8000, token.RemainQuota)
					assert.Equal(t, 2000, token.UsedQuota)
					assert.Equal(t, float64(123), stored.PrivateData.ImageAsync.ImageQuotaBeforeGroup)
					assert.Equal(t, 2000, info.Billing.GetPreConsumedQuota())
					require.Nil(t, PrepareImageBillingForRequest(c, info, 2), "retry must not reserve the quantity twice")
					require.NoError(t, db.First(&token, token.Id).Error)
					assert.Equal(t, 8000, token.RemainQuota)
				}
				if source == BillingSourceSubscription {
					require.NoError(t, db.First(&sub, sub.Id).Error)
					want := int64(2000)
					if rejectCheckpoint {
						want = 1000
					}
					assert.Equal(t, want, sub.AmountUsed)
					assert.Equal(t, 9000, user.Quota)
				} else if !rejectCheckpoint {
					assert.Equal(t, 8000, user.Quota)
				}
			})
		}
	}
}

func TestImageAsyncReservationRejectsPreviousWorkerAfterTaskReclaimed(t *testing.T) {
	db := setupImageAsyncBillingDB(t)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	user := model.User{Username: "reclaimed", Quota: 9000}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "reclaimed", RemainQuota: 9000, UsedQuota: 1000}
	require.NoError(t, db.Create(&token).Error)
	task := &model.Task{TaskID: "reclaimed", UserId: user.Id, Status: model.TaskStatusInProgress, Quota: 1000, PrivateData: model.TaskPrivateData{TokenId: token.Id, ImageAsync: &model.ImageAsyncPrivateData{Attempt: 2, WorkerNode: "current", ImageQuotaBeforeGroup: 321}}}
	require.NoError(t, db.Create(task).Error)
	for _, oldWorker := range []model.ImageAsyncPrivateData{{Attempt: 1, WorkerNode: "current"}, {Attempt: 2, WorkerNode: "previous"}} {
		stale := *task
		stale.PrivateData.ImageAsync = &oldWorker
		reserved, err := model.ReserveImageAsyncReservation(&stale, 2000)
		require.NoError(t, err)
		assert.False(t, reserved)
	}
	var stored model.Task
	require.NoError(t, db.First(&stored, task.ID).Error)
	require.NoError(t, db.First(&user, user.Id).Error)
	require.NoError(t, db.First(&token, token.Id).Error)
	assert.Equal(t, 1000, stored.Quota)
	assert.Equal(t, float64(321), stored.PrivateData.ImageAsync.ImageQuotaBeforeGroup)
	assert.Equal(t, 9000, user.Quota)
	assert.Equal(t, 9000, token.RemainQuota)
}

func TestFinalizeImageAsyncFailureRefundPreservesUsageAndSurvivesReload(t *testing.T) {
	db := setupImageAsyncBillingDB(t)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}))
	user := model.User{Username: "image_refund_stats", Quota: 9000, UsedQuota: 700, RequestCount: 3}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "image-refund-stats", RemainQuota: 9000, UsedQuota: 1000}
	require.NoError(t, db.Create(&token).Error)
	channel := model.Channel{Name: "image_refund_stats", UsedQuota: 700}
	require.NoError(t, db.Create(&channel).Error)
	task := &model.Task{TaskID: "image_refund_stats", UserId: user.Id, ChannelId: channel.Id, Quota: 1000, Status: model.TaskStatusFailure, PrivateData: model.TaskPrivateData{BillingSource: BillingSourceWallet, TokenId: token.Id, ImageAsync: &model.ImageAsyncPrivateData{}}}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, FinalizeImageAsyncFailureRefund(context.Background(), task, "upstream failed"))
	var reloaded model.Task
	require.NoError(t, db.First(&reloaded, task.ID).Error)
	require.NoError(t, FinalizeImageAsyncFailureRefund(context.Background(), &reloaded, "retry"))
	require.NoError(t, db.First(&user, user.Id).Error)
	require.NoError(t, db.First(&token, token.Id).Error)
	require.NoError(t, db.First(&channel, channel.Id).Error)
	require.Equal(t, 10000, user.Quota)
	require.Equal(t, 10000, token.RemainQuota)
	require.Equal(t, 0, token.UsedQuota)
	require.Equal(t, 700, user.UsedQuota, "pre-consumption has not incremented user usage")
	require.Equal(t, int64(700), channel.UsedQuota, "pre-consumption has not incremented channel usage")
	require.Equal(t, 3, user.RequestCount)
}

func TestFinalizeImageAsyncFailureRefundKeepsRetryableStateOnFundingFailure(t *testing.T) {
	db := setupImageAsyncBillingDB(t)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	task := &model.Task{
		TaskID: "image_async_failed_refund",
		UserId: 7,
		Quota:  1000,
		Status: model.TaskStatusFailure,
		PrivateData: model.TaskPrivateData{
			BillingSource:  BillingSourceSubscription,
			SubscriptionId: 99,
			ImageAsync:     &model.ImageAsyncPrivateData{},
		},
	}
	require.NoError(t, db.Create(task).Error)
	require.Error(t, FinalizeImageAsyncFailureRefund(context.Background(), task, "upstream failed"))
	var stored model.Task
	require.NoError(t, db.First(&stored, task.ID).Error)
	require.False(t, stored.PrivateData.ImageAsync.BillingRefunded)
	require.Equal(t, 1000, stored.Quota)
}
