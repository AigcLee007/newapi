package visionary

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type ImageRequest struct {
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
	Ratio     string `json:"ratio,omitempty"`
	ImageSize string `json:"imageSize,omitempty"`
	Quality   string `json:"quality,omitempty"`
	Images    any    `json:"images,omitempty"`
}

type ImageResponse struct {
	ID           string        `json:"id"`
	Results      []ImageResult `json:"results"`
	Error        any           `json:"error,omitempty"`
	ResponseID   string        `json:"responseId,omitempty"`
	ModelVersion string        `json:"modelVersion,omitempty"`
	CreatedAt    int64         `json:"createdAt,omitempty"`
}

type ImageResult struct {
	URL     string `json:"url"`
	Content string `json:"content,omitempty"`
}

func oaiImage2VisionaryImageRequest(request dto.ImageRequest) (*ImageRequest, error) {
	imageRequest := &ImageRequest{
		Prompt:  request.Prompt,
		Model:   request.Model,
		Ratio:   visionaryRatioFromImageRequest(request),
		Quality: request.Quality,
	}
	if imageRequest.Model == "" {
		imageRequest.Model = "gpt-image-2-official"
	}
	if imageSize := visionaryImageSizeFromImageRequest(request); imageSize != "" {
		imageRequest.ImageSize = imageSize
	}
	if len(request.Images) > 0 && string(request.Images) != "null" {
		var images any
		if err := common.Unmarshal(request.Images, &images); err != nil {
			return nil, fmt.Errorf("invalid images field: %w", err)
		}
		imageRequest.Images = images
	}
	return imageRequest, nil
}

func visionaryRatioFromImageRequest(request dto.ImageRequest) string {
	if ratio := stringExtra(request, "ratio"); ratio != "" {
		return ratio
	}
	if aspectRatio := stringExtra(request, "aspect_ratio"); aspectRatio != "" {
		return aspectRatio
	}
	if request.Size == "" {
		return ""
	}
	if strings.Contains(request.Size, "x") {
		return request.Size
	}
	return request.Size
}

func visionaryImageSizeFromImageRequest(request dto.ImageRequest) string {
	if imageSize := stringExtra(request, "imageSize"); imageSize != "" {
		return imageSize
	}
	if imageSize := stringExtra(request, "image_size"); imageSize != "" {
		return imageSize
	}
	return ""
}

func stringExtra(request dto.ImageRequest, key string) string {
	raw, ok := request.Extra[key]
	if !ok || len(raw) == 0 {
		return ""
	}
	var value string
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func responseVisionary2OpenAIImage(response *ImageResponse, info *relaycommon.RelayInfo) *dto.ImageResponse {
	imageResponse := &dto.ImageResponse{
		Created: info.StartTime.Unix(),
	}
	for _, result := range response.Results {
		if result.URL != "" {
			imageResponse.Data = append(imageResponse.Data, dto.ImageData{Url: result.URL})
		}
	}
	return imageResponse
}

func visionaryImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	var visionaryResponse ImageResponse
	if err := common.Unmarshal(responseBody, &visionaryResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if visionaryResponse.Error != nil {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: fmt.Sprintf("%v", visionaryResponse.Error),
			Type:    "visionary_image_error",
			Code:    "visionary_image_error",
		}, resp.StatusCode)
	}

	openAIResponse := responseVisionary2OpenAIImage(&visionaryResponse, info)
	jsonResponse, err := common.Marshal(openAIResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err := c.Writer.Write(jsonResponse); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	imageCount := len(openAIResponse.Data)
	if imageCount > 0 {
		info.PriceData.AddOtherRatio("n", float64(imageCount))
	}
	return &dto.Usage{}, nil
}
