package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
