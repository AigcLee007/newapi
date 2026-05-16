package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
