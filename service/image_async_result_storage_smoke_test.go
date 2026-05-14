//go:build smoke

package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestImageAsyncRealCOSSmoke(t *testing.T) {
	required := []string{
		"IMAGE_ASYNC_S3_ACCESS_KEY_ID",
		"IMAGE_ASYNC_S3_SECRET_ACCESS_KEY",
	}
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			t.Skipf("%s is not set", key)
		}
	}

	t.Setenv("IMAGE_ASYNC_RESULT_STORAGE", "s3")
	t.Setenv("IMAGE_ASYNC_S3_PROVIDER", "tencent-cos")
	t.Setenv("IMAGE_ASYNC_S3_ENDPOINT", "https://cos.ap-hongkong.myqcloud.com")
	t.Setenv("IMAGE_ASYNC_S3_REGION", "ap-hongkong")
	t.Setenv("IMAGE_ASYNC_S3_BUCKET", "lhcos-8330f-1430163992")
	t.Setenv("IMAGE_ASYNC_S3_PUBLIC_BASE_URL", "https://lhcos-8330f-1430163992.cos.ap-hongkong.myqcloud.com")
	t.Setenv("IMAGE_ASYNC_S3_FORCE_PATH_STYLE", "false")
	t.Setenv("IMAGE_ASYNC_S3_OBJECT_PREFIX", "newapi/images/smoke/")
	t.Setenv("IMAGE_ASYNC_CONVERT_B64_TO_URL", "true")
	t.Setenv("IMAGE_ASYNC_CACHE_UPSTREAM_URL", "false")
	t.Setenv("IMAGE_ASYNC_RESULT_MAX_MB", "1")

	task := &model.Task{
		TaskID: "task_cos_smoke",
		PrivateData: model.TaskPrivateData{
			ImageAsync: &model.ImageAsyncPrivateData{},
		},
	}
	body := []byte(fmt.Sprintf(`{"data":[{"b64_json":%q,"revised_prompt":"cos smoke test"}],"model":"test-image-model","created":1710000000,"usage":{}}`, imageAsyncTinyPNGBase64))

	normalized, err := NormalizeImageAsyncResultStorage(context.Background(), task, body)
	require.NoError(t, err)
	require.Len(t, task.PrivateData.ImageAsync.ResultObjects, 1)
	t.Cleanup(func() {
		CleanupImageAsyncResultObjects(context.Background(), task)
	})

	var parsed struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(normalized, &parsed))
	require.Len(t, parsed.Data, 1)
	require.Empty(t, parsed.Data[0].B64JSON)
	require.True(t, strings.HasPrefix(parsed.Data[0].URL, "https://lhcos-8330f-1430163992.cos.ap-hongkong.myqcloud.com/newapi/images/smoke/"))

	resp, err := http.Get(parsed.Data[0].URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "image/"))
	require.Greater(t, resp.ContentLength, int64(0))
}
