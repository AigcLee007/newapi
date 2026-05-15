package visionary

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestOAIImage2VisionaryImageRequest(t *testing.T) {
	request := dto.ImageRequest{
		Model:   "gpt-image-2-official",
		Prompt:  "city at night",
		Size:    "16:9",
		Quality: "auto",
		Images:  []byte(`["https://example.com/reference.png"]`),
		Extra: map[string]json.RawMessage{
			"imageSize": json.RawMessage(`"2K"`),
		},
	}

	got, err := oaiImage2VisionaryImageRequest(request)
	if err != nil {
		t.Fatalf("oaiImage2VisionaryImageRequest returned error: %v", err)
	}
	if got.Model != "gpt-image-2-official" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.Prompt != "city at night" {
		t.Fatalf("prompt = %q", got.Prompt)
	}
	if got.Ratio != "16:9" {
		t.Fatalf("ratio = %q", got.Ratio)
	}
	if got.ImageSize != "2K" {
		t.Fatalf("imageSize = %q", got.ImageSize)
	}
	if got.Images == nil {
		t.Fatal("images should be preserved")
	}
}

func TestResponseVisionary2OpenAIImage(t *testing.T) {
	response := &ImageResponse{
		Results: []ImageResult{
			{URL: "https://visionary.beer/openapi-assets/example-result.png"},
		},
	}
	info := &relaycommon.RelayInfo{}

	got := responseVisionary2OpenAIImage(response, info)
	if len(got.Data) != 1 {
		t.Fatalf("data length = %d", len(got.Data))
	}
	if got.Data[0].Url != response.Results[0].URL {
		t.Fatalf("url = %q", got.Data[0].Url)
	}
}
