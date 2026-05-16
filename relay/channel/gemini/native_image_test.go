package gemini

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestOAIImage2GeminiNativeImageRequest(t *testing.T) {
	req := dto.ImageRequest{
		Model:  "gemini-3-pro-image-preview",
		Prompt: "Keep the composition, make it watercolor style",
		Images: json.RawMessage(`["https://example.com/input.jpg"]`),
		Extra: map[string]json.RawMessage{
			"aspect_ratio": json.RawMessage(`"16:9"`),
			"imageSize":    json.RawMessage(`"2K"`),
		},
	}

	got, err := oaiImage2GeminiNativeImageRequest(nil, req)
	if err != nil {
		t.Fatalf("oaiImage2GeminiNativeImageRequest returned error: %v", err)
	}
	if len(got.Contents) != 1 || len(got.Contents[0].Parts) != 2 {
		t.Fatalf("unexpected contents: %+v", got.Contents)
	}
	if got.GenerationConfig.ResponseModalities[0] != "IMAGE" {
		t.Fatalf("response modalities = %+v", got.GenerationConfig.ResponseModalities)
	}
	var imageConfig map[string]string
	if err := json.Unmarshal(got.GenerationConfig.ImageConfig, &imageConfig); err != nil {
		t.Fatalf("invalid image config: %v", err)
	}
	if imageConfig["aspectRatio"] != "16:9" || imageConfig["imageSize"] != "2K" {
		t.Fatalf("image config = %+v", imageConfig)
	}
	if got.Contents[0].Parts[1].FileData == nil || got.Contents[0].Parts[1].FileData.FileUri != "https://example.com/input.jpg" {
		t.Fatalf("file data = %+v", got.Contents[0].Parts[1].FileData)
	}
}
