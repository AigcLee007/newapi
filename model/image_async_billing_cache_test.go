package model

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupImageAsyncCacheReservation(t *testing.T) (*Task, User, Token) {
	t.Helper()
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)
	common.BatchUpdateEnabled = true
	user := createReserveTestUser(t, 100)
	token := createReserveTestToken(t, 100)
	token.UserId = user.Id
	require.NoError(t, DB.Model(&token).Update("user_id", user.Id).Error)
	task := &Task{TaskID: "image-cache-" + common.GetRandomString(8), UserId: user.Id, Status: TaskStatusInProgress, PrivateData: TaskPrivateData{TokenId: token.Id, BillingSource: "wallet", ImageAsync: &ImageAsyncPrivateData{}}}
	require.NoError(t, DB.Create(task).Error)
	return task, user, token
}

func requireImageAsyncCachedQuota(t *testing.T, user User, token Token, quota int) {
	t.Helper()
	cachedUser, err := GetUserCache(user.Id)
	require.NoError(t, err)
	cachedToken, err := cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	require.Equal(t, quota, cachedUser.Quota)
	require.Equal(t, quota, cachedToken.RemainQuota)
}

func TestImageAsyncReservationRespectsPendingBatchSpending(t *testing.T) {
	for _, quota := range []int{10, 30} {
		t.Run(common.Interface2String(quota), func(t *testing.T) {
			task, user, token := setupImageAsyncCacheReservation(t)
			ok, err := TryReserveUserQuota(user.Id, 80)
			require.NoError(t, err)
			require.True(t, ok)
			ok, err = TryReserveTokenQuota(token.Id, token.Key, 80, false)
			require.NoError(t, err)
			require.True(t, ok)
			ok, err = ReserveImageAsyncReservation(task, quota)
			if quota > 20 {
				require.False(t, ok)
				requireImageAsyncCachedQuota(t, user, token, 20)
				require.Equal(t, 100, getUserQuotaFromDB(t, user.Id))
			} else {
				require.NoError(t, err)
				require.True(t, ok)
				requireImageAsyncCachedQuota(t, user, token, 10)
				ok, err = ReserveImageAsyncReservation(task, quota)
				require.NoError(t, err)
				require.True(t, ok)
				requireImageAsyncCachedQuota(t, user, token, 10)
				batchUpdate()
				require.Equal(t, 10, getUserQuotaFromDB(t, user.Id))
				require.Equal(t, 10, getTokenFromDB(t, token.Id).RemainQuota)
			}
		})
	}
}

func TestImageAsyncReservationCacheRollbackPreservesPendingBatch(t *testing.T) {
	task, user, token := setupImageAsyncCacheReservation(t)
	ok, err := TryReserveUserQuota(user.Id, 20)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = TryReserveTokenQuota(token.Id, token.Key, 20, false)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register("reject-image-cache-checkpoint", func(tx *gorm.DB) {
		if tx.Statement.Table == "tasks" {
			tx.AddError(errors.New("checkpoint rejected"))
		}
	}))
	t.Cleanup(func() { DB.Callback().Update().Remove("reject-image-cache-checkpoint") })
	ok, err = ReserveImageAsyncReservation(task, 10)
	require.Error(t, err)
	require.False(t, ok)
	requireImageAsyncCachedQuota(t, user, token, 80)
	require.Equal(t, 100, getUserQuotaFromDB(t, user.Id))
}

func TestImageAsyncRefundCachePreservesPendingBatchAndAvoidsRehydrationCredit(t *testing.T) {
	for _, hydrate := range []bool{false, true} {
		t.Run(common.Interface2String(hydrate), func(t *testing.T) {
			task, user, token := setupImageAsyncCacheReservation(t)
			ok, err := ReserveImageAsyncReservation(task, 20)
			require.NoError(t, err)
			require.True(t, ok)
			ok, err = TryReserveUserQuota(user.Id, 30)
			require.NoError(t, err)
			require.True(t, ok)
			ok, err = TryReserveTokenQuota(token.Id, token.Key, 30, false)
			require.NoError(t, err)
			require.True(t, ok)
			if hydrate {
				batchUpdate()
				require.NoError(t, DB.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("hydrate-image-refund", func(tx *gorm.DB) {
					if tx.Statement.Table != "tasks" {
						return
					}
					require.NoError(t, common.RDB.Del(context.Background(), getUserCacheKey(user.Id), getTokenCacheKey(token.Key)).Err())
					var freshUser User
					var freshToken Token
					require.NoError(t, tx.Session(&gorm.Session{NewDB: true}).First(&freshUser, user.Id).Error)
					require.NoError(t, tx.Session(&gorm.Session{NewDB: true}).First(&freshToken, token.Id).Error)
					require.NoError(t, populateUserCache(freshUser))
					_, err := cacheInitToken(freshToken)
					require.NoError(t, err)
				}))
				t.Cleanup(func() { DB.Callback().Update().Remove("hydrate-image-refund") })
			}
			ok, err = RefundImageAsyncReservation(task)
			require.NoError(t, err)
			require.True(t, ok)
			requireImageAsyncCachedQuota(t, user, token, 70)
			batchUpdate()
			require.Equal(t, 70, getUserQuotaFromDB(t, user.Id))
			require.Equal(t, 70, getTokenFromDB(t, token.Id).RemainQuota)
		})
	}
}

func TestImageAsyncCacheCheckpointCompensatesOnlyItsOriginalHash(t *testing.T) {
	for _, replaceHash := range []bool{false, true} {
		t.Run(common.Interface2String(replaceHash), func(t *testing.T) {
			_, user, token := setupImageAsyncCacheReservation(t)
			_, err := GetUserCache(user.Id)
			require.NoError(t, err)
			_, err = GetTokenByKey(token.Key, false)
			require.NoError(t, err)
			checkpoint := newImageAsyncCacheCheckpoint()
			require.NoError(t, checkpoint.apply(user.Id, getUserCacheKey(user.Id), true, 10, true, false))
			require.NoError(t, checkpoint.apply(token.Id, getTokenCacheKey(token.Key), false, 10, true, false))
			if replaceHash {
				require.NoError(t, common.RDB.Del(context.Background(), getUserCacheKey(user.Id), getTokenCacheKey(token.Key)).Err())
				require.NoError(t, populateUserCache(user))
				_, err := cacheInitToken(token)
				require.NoError(t, err)
			}
			checkpoint.finish(false)
			checkpoint.finish(false)
			requireImageAsyncCachedQuota(t, user, token, 100)
		})
	}
}

func TestImageAsyncCacheCheckpointKeepsConcurrentTaskMarkersSeparate(t *testing.T) {
	_, user, token := setupImageAsyncCacheReservation(t)
	_, err := GetUserCache(user.Id)
	require.NoError(t, err)
	_, err = GetTokenByKey(token.Key, false)
	require.NoError(t, err)
	first, second := newImageAsyncCacheCheckpoint(), newImageAsyncCacheCheckpoint()
	for _, checkpoint := range []*imageAsyncCacheCheckpoint{first, second} {
		require.NoError(t, checkpoint.apply(user.Id, getUserCacheKey(user.Id), true, 10, false, false))
		require.NoError(t, checkpoint.apply(token.Id, getTokenCacheKey(token.Key), false, 10, false, false))
	}
	first.finish(true)
	second.finish(true)
	first.finish(true)
	requireImageAsyncCachedQuota(t, user, token, 120)
}

func TestImageAsyncReservationFailsClosedWhenQuotaCacheDisappears(t *testing.T) {
	task, user, token := setupImageAsyncCacheReservation(t)
	ok, err := TryReserveUserQuota(user.Id, 80)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = TryReserveTokenQuota(token.Id, token.Key, 80, false)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, DB.Callback().Query().After("gorm:query").Register("delete-image-quota-cache", func(tx *gorm.DB) {
		if tx.Statement.Table == "tasks" {
			require.NoError(t, common.RDB.Del(context.Background(), getUserCacheKey(user.Id), getTokenCacheKey(token.Key)).Err())
		}
	}))
	t.Cleanup(func() { DB.Callback().Query().Remove("delete-image-quota-cache") })
	ok, err = ReserveImageAsyncReservation(task, 10)
	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, 100, getUserQuotaFromDB(t, user.Id))
}
