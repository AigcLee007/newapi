package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageAsyncRequestBodyStorageRoundTripAndDelete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IMAGE_ASYNC_STORAGE_DIR", dir)
	t.Setenv("IMAGE_ASYNC_MAX_BODY_MB", "1")

	path, sha, size, err := SaveImageAsyncRequestBody("task_storage_1", []byte("hello image async"))
	require.NoError(t, err)
	assert.EqualValues(t, len("hello image async"), size)
	assert.True(t, strings.HasPrefix(path, dir))

	data, err := ReadImageAsyncRequestBody(path, sha, 1024)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello image async"), data)

	require.NoError(t, DeleteImageAsyncRequestBody(path))
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestImageAsyncRequestBodyStorageRejectsUnsafePathAndBadChecksum(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IMAGE_ASYNC_STORAGE_DIR", dir)

	_, _, _, err := SaveImageAsyncRequestBody("../escape", []byte("x"))
	require.Error(t, err)

	path, _, _, err := SaveImageAsyncRequestBody("task_storage_2", []byte("body"))
	require.NoError(t, err)
	_, err = ReadImageAsyncRequestBody(path, "bad-sha", 1024)
	require.Error(t, err)

	outside := filepath.Join(t.TempDir(), "request.body")
	require.NoError(t, os.WriteFile(outside, []byte("outside"), 0600))
	_, err = ReadImageAsyncRequestBody(outside, "", 1024)
	require.Error(t, err)
}

func TestCleanupExpiredImageAsyncBodiesOnlyRemovesExpiredBodyFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IMAGE_ASYNC_STORAGE_DIR", dir)

	oldPath, _, _, err := SaveImageAsyncRequestBody("task_old", []byte("old"))
	require.NoError(t, err)
	newPath, _, _, err := SaveImageAsyncRequestBody("task_new", []byte("new"))
	require.NoError(t, err)
	keepPath := filepath.Join(filepath.Dir(oldPath), "metadata.json")
	require.NoError(t, os.WriteFile(keepPath, []byte("{}"), 0600))

	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldPath, oldTime, oldTime))
	require.NoError(t, os.Chtimes(keepPath, oldTime, oldTime))

	require.NoError(t, CleanupExpiredImageAsyncBodies(time.Hour))

	_, err = os.Stat(oldPath)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(newPath)
	assert.NoError(t, err)
	_, err = os.Stat(keepPath)
	assert.NoError(t, err)
}
