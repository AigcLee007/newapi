package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func TestOpenAIToGeminiNormalizesInlineImageDataURL(t *testing.T) {
	request := dto.GeneralOpenAIRequest{Model: "gemini-2.5-flash", Messages: []dto.Message{{Role: "user", Content: []dto.MediaContent{{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "base64:data:image/jpeg;base64,_9j_4AAQ SkZJRgABAQAAAQABAAD_2w"}}}}}}
	converted, err := relayconvert.OpenAIChatRequestToGeminiGenerateContent(context.Background(), request, &convmeta.Values{OriginModelName: request.Model})
	require.NoError(t, err)
	require.Len(t, converted.Contents, 1)
	require.Len(t, converted.Contents[0].Parts, 1)
	require.Equal(t, "image/jpeg", converted.Contents[0].Parts[0].InlineData.MimeType)
	require.Equal(t, "/9j/4AAQSkZJRgABAQAAAQABAAD/2w==", converted.Contents[0].Parts[0].InlineData.Data)
}

func TestOpenAIToGeminiLoadsRemoteImageThroughResolver(t *testing.T) {
	service.InitHttpClient()
	previousMaxFileSize := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	t.Cleanup(func() { constant.MaxFileDownloadMB = previousMaxFileSize })
	settings := system_setting.GetFetchSetting()
	original := *settings
	settings.EnableSSRFProtection = false
	t.Cleanup(func() { *settings = original })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("image bytes"))
	}))
	defer server.Close()
	request := dto.GeneralOpenAIRequest{Model: "gemini-2.5-flash", Messages: []dto.Message{{Role: "user", Content: []dto.MediaContent{{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: server.URL}}}}}}
	converted, err := relayconvert.OpenAIChatRequestToGeminiGenerateContent(context.Background(), request, &convmeta.Values{OriginModelName: request.Model})
	require.NoError(t, err)
	require.Len(t, converted.Contents, 1)
	require.Len(t, converted.Contents[0].Parts, 1)
	require.Equal(t, "image/png", converted.Contents[0].Parts[0].InlineData.MimeType)
	require.Equal(t, "aW1hZ2UgYnl0ZXM=", converted.Contents[0].Parts[0].InlineData.Data)
}
