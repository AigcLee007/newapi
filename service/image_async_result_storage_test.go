package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const imageAsyncTinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="

type mockImageAsyncUploader struct {
	puts    []mockImageAsyncPut
	deletes []string
	err     error
}

type mockImageAsyncPut struct {
	key         string
	body        []byte
	size        int64
	contentType string
}

func (m *mockImageAsyncUploader) PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if m.err != nil {
		return m.err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.puts = append(m.puts, mockImageAsyncPut{
		key:         key,
		body:        data,
		size:        size,
		contentType: contentType,
	})
	return nil
}

func (m *mockImageAsyncUploader) DeleteObject(ctx context.Context, key string) error {
	m.deletes = append(m.deletes, key)
	return nil
}

type mockImageAsyncDownloader struct {
	result *ImageAsyncDownloadedObject
	err    error
	calls  int
}

func (m *mockImageAsyncDownloader) Download(ctx context.Context, sourceURL string, maxBytes int64) (*ImageAsyncDownloadedObject, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func setImageAsyncResultStorageTestHooks(t *testing.T, uploader ImageAsyncObjectUploader, downloader ImageAsyncResultDownloader) {
	t.Helper()
	imageAsyncResultStorageMu.Lock()
	oldUploader := imageAsyncResultUploaderOverride
	oldDownloader := imageAsyncResultDownloaderOverride
	oldNow := imageAsyncResultNow
	imageAsyncResultUploaderOverride = uploader
	imageAsyncResultDownloaderOverride = downloader
	imageAsyncResultNow = func() time.Time {
		return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	}
	imageAsyncResultStorageMu.Unlock()
	t.Cleanup(func() {
		imageAsyncResultStorageMu.Lock()
		imageAsyncResultUploaderOverride = oldUploader
		imageAsyncResultDownloaderOverride = oldDownloader
		imageAsyncResultNow = oldNow
		imageAsyncResultStorageMu.Unlock()
	})
}

func setImageAsyncS3Env(t *testing.T) {
	t.Helper()
	t.Setenv("IMAGE_ASYNC_RESULT_STORAGE", "s3")
	t.Setenv("IMAGE_ASYNC_S3_PROVIDER", "tencent-cos")
	t.Setenv("IMAGE_ASYNC_S3_ENDPOINT", "https://cos.ap-hongkong.myqcloud.com")
	t.Setenv("IMAGE_ASYNC_S3_REGION", "ap-hongkong")
	t.Setenv("IMAGE_ASYNC_S3_BUCKET", "lhcos-8330f-1430163992")
	t.Setenv("IMAGE_ASYNC_S3_ACCESS_KEY_ID", "test-secret-id")
	t.Setenv("IMAGE_ASYNC_S3_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("IMAGE_ASYNC_S3_PUBLIC_BASE_URL", "https://lhcos-8330f-1430163992.cos.ap-hongkong.myqcloud.com")
	t.Setenv("IMAGE_ASYNC_S3_FORCE_PATH_STYLE", "false")
	t.Setenv("IMAGE_ASYNC_S3_OBJECT_PREFIX", "newapi/images/")
	t.Setenv("IMAGE_ASYNC_CONVERT_B64_TO_URL", "true")
	t.Setenv("IMAGE_ASYNC_CACHE_UPSTREAM_URL", "true")
	t.Setenv("IMAGE_ASYNC_RESULT_MAX_MB", "1")
}

func newImageAsyncResultTask() *model.Task {
	return &model.Task{
		TaskID: "task_test",
		PrivateData: model.TaskPrivateData{
			ImageAsync: &model.ImageAsyncPrivateData{},
		},
	}
}

func TestNormalizeImageAsyncResultStoragePassthroughKeepsBody(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_RESULT_STORAGE", "passthrough")
	uploader := &mockImageAsyncUploader{}
	setImageAsyncResultStorageTestHooks(t, uploader, nil)
	body := []byte(`{"data":[{"url":"https://upstream.example.com/output.png","b64_json":"keep"}],"model":"m"}`)

	normalized, err := NormalizeImageAsyncResultStorage(context.Background(), newImageAsyncResultTask(), body)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(body, normalized))
	assert.Empty(t, uploader.puts)
}

func TestNormalizeImageAsyncResultStorageUploadsB64JSON(t *testing.T) {
	setImageAsyncS3Env(t)
	uploader := &mockImageAsyncUploader{}
	setImageAsyncResultStorageTestHooks(t, uploader, nil)
	body := []byte(fmt.Sprintf(`{"data":[{"b64_json":%q,"revised_prompt":"cat","unknown":true}],"model":"m","created":123,"usage":{"total_tokens":1},"top_unknown":"keep"}`, imageAsyncTinyPNGBase64))
	task := newImageAsyncResultTask()

	normalized, err := NormalizeImageAsyncResultStorage(context.Background(), task, body)
	require.NoError(t, err)
	require.Len(t, uploader.puts, 1)
	assert.Equal(t, "newapi/images/2026/05/14/task_test_0.png", uploader.puts[0].key)
	assert.Equal(t, "image/png", uploader.puts[0].contentType)
	assert.EqualValues(t, len(uploader.puts[0].body), uploader.puts[0].size)

	var parsed map[string]any
	require.NoError(t, common.Unmarshal(normalized, &parsed))
	data := parsed["data"].([]any)
	item := data[0].(map[string]any)
	assert.Equal(t, "https://lhcos-8330f-1430163992.cos.ap-hongkong.myqcloud.com/newapi/images/2026/05/14/task_test_0.png", item["url"])
	assert.Equal(t, "", item["b64_json"])
	assert.Equal(t, "cat", item["revised_prompt"])
	assert.Equal(t, true, item["unknown"])
	assert.Equal(t, "m", parsed["model"])
	assert.Equal(t, "keep", parsed["top_unknown"])
	require.Len(t, task.PrivateData.ImageAsync.ResultObjects, 1)
	assert.Equal(t, imageAsyncResultSourceB64, task.PrivateData.ImageAsync.ResultObjects[0].Source)
}

func TestNormalizeImageAsyncResultStorageUploadsMultipleImages(t *testing.T) {
	setImageAsyncS3Env(t)
	uploader := &mockImageAsyncUploader{}
	setImageAsyncResultStorageTestHooks(t, uploader, nil)
	body := []byte(fmt.Sprintf(`{"data":[{"b64_json":%q},{"b64_json":%q}]}`, imageAsyncTinyPNGBase64, imageAsyncTinyPNGBase64))

	normalized, err := NormalizeImageAsyncResultStorage(context.Background(), newImageAsyncResultTask(), body)
	require.NoError(t, err)
	require.Len(t, uploader.puts, 2)
	assert.Equal(t, "newapi/images/2026/05/14/task_test_0.png", uploader.puts[0].key)
	assert.Equal(t, "newapi/images/2026/05/14/task_test_1.png", uploader.puts[1].key)
	assert.Contains(t, string(normalized), "task_test_0.png")
	assert.Contains(t, string(normalized), "task_test_1.png")
}

func TestNormalizeImageAsyncResultStorageUpstreamURLPassthroughWhenCacheDisabled(t *testing.T) {
	setImageAsyncS3Env(t)
	t.Setenv("IMAGE_ASYNC_CACHE_UPSTREAM_URL", "false")
	uploader := &mockImageAsyncUploader{}
	downloader := &mockImageAsyncDownloader{}
	setImageAsyncResultStorageTestHooks(t, uploader, downloader)
	body := []byte(`{"data":[{"url":"https://upstream.example.com/output.png","revised_prompt":"cat"}]}`)

	normalized, err := NormalizeImageAsyncResultStorage(context.Background(), newImageAsyncResultTask(), body)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(body, normalized))
	assert.Empty(t, uploader.puts)
	assert.Equal(t, 0, downloader.calls)
}

func TestNormalizeImageAsyncResultStorageCachesUpstreamURL(t *testing.T) {
	setImageAsyncS3Env(t)
	png, _, _, err := decodeImageAsyncB64JSON(imageAsyncTinyPNGBase64, 1<<20)
	require.NoError(t, err)
	uploader := &mockImageAsyncUploader{}
	downloader := &mockImageAsyncDownloader{result: &ImageAsyncDownloadedObject{
		Body:        png,
		ContentType: "image/png",
		Ext:         "png",
		Size:        int64(len(png)),
	}}
	setImageAsyncResultStorageTestHooks(t, uploader, downloader)
	body := []byte(`{"data":[{"url":"https://upstream.example.com/output.png","revised_prompt":"cat"}],"model":"m"}`)

	normalized, err := NormalizeImageAsyncResultStorage(context.Background(), newImageAsyncResultTask(), body)
	require.NoError(t, err)
	require.Len(t, uploader.puts, 1)
	assert.Equal(t, 1, downloader.calls)
	assert.Contains(t, string(normalized), "lhcos-8330f-1430163992.cos.ap-hongkong.myqcloud.com/newapi/images/2026/05/14/task_test_0.png")
	assert.Contains(t, string(normalized), "revised_prompt")
}

func TestNormalizeImageAsyncResultStorageMissingS3Config(t *testing.T) {
	required := []string{
		"IMAGE_ASYNC_S3_ENDPOINT",
		"IMAGE_ASYNC_S3_BUCKET",
		"IMAGE_ASYNC_S3_ACCESS_KEY_ID",
		"IMAGE_ASYNC_S3_SECRET_ACCESS_KEY",
		"IMAGE_ASYNC_S3_PUBLIC_BASE_URL",
	}
	for _, missing := range required {
		t.Run(missing, func(t *testing.T) {
			setImageAsyncS3Env(t)
			t.Setenv(missing, "")
			setImageAsyncResultStorageTestHooks(t, nil, nil)
			body := []byte(fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, imageAsyncTinyPNGBase64))
			_, err := NormalizeImageAsyncResultStorage(context.Background(), newImageAsyncResultTask(), body)
			require.Error(t, err)
			assert.Contains(t, err.Error(), missing)
		})
	}
}

func TestNormalizeImageAsyncResultStorageRejectsUnsafeAndInvalidImages(t *testing.T) {
	t.Run("localhost url", func(t *testing.T) {
		err := validateImageAsyncPublicURL("http://127.0.0.1/output.png")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private IP")
	})
	t.Run("private url", func(t *testing.T) {
		err := validateImageAsyncPublicURL("http://10.0.0.1/output.png")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private IP")
	})
	t.Run("invalid base64", func(t *testing.T) {
		_, _, _, err := decodeImageAsyncB64JSON("not-base64", 1<<20)
		require.Error(t, err)
	})
	t.Run("not image", func(t *testing.T) {
		_, _, _, err := decodeImageAsyncB64JSON("bm90LWltYWdl", 1<<20)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
	t.Run("too large", func(t *testing.T) {
		_, _, _, err := decodeImageAsyncB64JSON(imageAsyncTinyPNGBase64, 8)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds")
	})
	t.Run("content type mismatch", func(t *testing.T) {
		assert.False(t, imageAsyncContentTypesCompatible("text/plain", "image/png"))
		assert.False(t, imageAsyncContentTypesCompatible("image/jpeg", "image/png"))
		assert.True(t, imageAsyncContentTypesCompatible("image/png; charset=binary", "image/png"))
	})
}

func TestNormalizeImageAsyncResultStorageCleansUploadedObjectsOnError(t *testing.T) {
	setImageAsyncS3Env(t)
	uploader := &mockImageAsyncUploader{}
	downloader := &mockImageAsyncDownloader{err: fmt.Errorf("download failed")}
	setImageAsyncResultStorageTestHooks(t, uploader, downloader)
	body := []byte(fmt.Sprintf(`{"data":[{"b64_json":%q},{"url":"https://upstream.example.com/output.png"}]}`, imageAsyncTinyPNGBase64))

	_, err := NormalizeImageAsyncResultStorage(context.Background(), newImageAsyncResultTask(), body)
	require.Error(t, err)
	require.Len(t, uploader.puts, 1)
	assert.Equal(t, []string{"newapi/images/2026/05/14/task_test_0.png"}, uploader.deletes)
}

func TestNormalizeImageAsyncObjectPrefixRejectsTraversal(t *testing.T) {
	_, err := normalizeImageAsyncObjectPrefix("../images")
	require.Error(t, err)
	_, err = normalizeImageAsyncObjectPrefix("newapi/../images")
	require.Error(t, err)
	prefix, err := normalizeImageAsyncObjectPrefix("/newapi/images/")
	require.NoError(t, err)
	assert.Equal(t, "newapi/images/", prefix)
	assert.False(t, strings.Contains(prefix, ".."))
}
