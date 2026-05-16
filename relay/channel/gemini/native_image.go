package gemini

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func isGeminiNativeImageModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(modelName, "gemini-") && strings.Contains(modelName, "image")
}

func oaiImage2GeminiNativeImageRequest(c *gin.Context, request dto.ImageRequest) (*dto.GeminiChatRequest, error) {
	parts := []dto.GeminiPart{}
	if strings.TrimSpace(request.Prompt) != "" {
		parts = append(parts, dto.GeminiPart{Text: request.Prompt})
	}

	imageURLs := collectGeminiImageURLs(request)
	for _, imageURL := range imageURLs {
		parts = append(parts, dto.GeminiPart{
			FileData: &dto.GeminiFileData{
				MimeType: guessGeminiImageMimeType(imageURL),
				FileUri:  imageURL,
			},
		})
	}

	inlineParts, err := collectGeminiInlineImageParts(c)
	if err != nil {
		return nil, err
	}
	parts = append(parts, inlineParts...)

	if len(parts) == 0 {
		return nil, fmt.Errorf("prompt or image is required")
	}

	imageConfig := make(map[string]string)
	if aspectRatio := geminiImageAspectRatio(request); aspectRatio != "" {
		imageConfig["aspectRatio"] = aspectRatio
	}
	if imageSize := geminiImageSize(request); imageSize != "" {
		imageConfig["imageSize"] = imageSize
	}

	var imageConfigRaw []byte
	if len(imageConfig) > 0 {
		imageConfigRaw, err = common.Marshal(imageConfig)
		if err != nil {
			return nil, fmt.Errorf("marshal image config: %w", err)
		}
	}

	return &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role:  "user",
				Parts: parts,
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ResponseModalities: []string{"IMAGE"},
			ImageConfig:        imageConfigRaw,
		},
	}, nil
}

func collectGeminiImageURLs(request dto.ImageRequest) []string {
	seen := map[string]bool{}
	result := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		result = append(result, value)
	}

	appendImageURLsFromRaw(request.Images, add)
	appendImageURLsFromRaw(request.Image, add)
	return result
}

func appendImageURLsFromRaw(raw []byte, add func(string)) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}

	var single string
	if err := common.Unmarshal(raw, &single); err == nil {
		add(single)
		return
	}

	var values []string
	if err := common.Unmarshal(raw, &values); err == nil {
		for _, value := range values {
			add(value)
		}
		return
	}

	var generic any
	if err := common.Unmarshal(raw, &generic); err != nil {
		return
	}
	appendImageURLsFromAny(generic, add)
}

func appendImageURLsFromAny(value any, add func(string)) {
	switch typed := value.(type) {
	case string:
		add(typed)
	case []any:
		for _, item := range typed {
			appendImageURLsFromAny(item, add)
		}
	case map[string]any:
		for _, key := range []string{"url", "image_url", "fileUri", "file_uri"} {
			if raw, ok := typed[key].(string); ok {
				add(raw)
				return
			}
		}
		if nested, ok := typed["image_url"]; ok {
			appendImageURLsFromAny(nested, add)
		}
		if nested, ok := typed["fileData"]; ok {
			appendImageURLsFromAny(nested, add)
		}
	}
}

func collectGeminiInlineImageParts(c *gin.Context) ([]dto.GeminiPart, error) {
	if c == nil {
		return nil, nil
	}
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		return nil, nil
	}
	if _, err := c.MultipartForm(); err != nil {
		return nil, fmt.Errorf("parse multipart image request: %w", err)
	}
	if c.Request.MultipartForm == nil {
		return nil, nil
	}

	parts := []dto.GeminiPart{}
	for _, fieldName := range []string{"image", "images"} {
		for _, fileHeader := range c.Request.MultipartForm.File[fieldName] {
			part, err := geminiInlinePartFromFile(fileHeader)
			if err != nil {
				return nil, err
			}
			parts = append(parts, part)
		}
	}
	return parts, nil
}

func geminiInlinePartFromFile(fileHeader *multipart.FileHeader) (dto.GeminiPart, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return dto.GeminiPart{}, fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return dto.GeminiPart{}, fmt.Errorf("read image file: %w", err)
	}
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(body)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return dto.GeminiPart{}, fmt.Errorf("uploaded file is not an image")
	}

	return dto.GeminiPart{
		InlineData: &dto.GeminiInlineData{
			MimeType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(body),
		},
	}, nil
}

func geminiImageAspectRatio(request dto.ImageRequest) string {
	for _, key := range []string{"aspect_ratio", "aspectRatio", "ratio"} {
		if value := imageStringExtra(request, key); value != "" {
			return value
		}
	}
	size := strings.TrimSpace(request.Size)
	switch size {
	case "256x256", "512x512", "1024x1024":
		return "1:1"
	case "1536x1024":
		return "3:2"
	case "1024x1536":
		return "2:3"
	case "1024x1792":
		return "9:16"
	case "1792x1024":
		return "16:9"
	}
	if strings.Contains(size, ":") {
		return size
	}
	return ""
}

func geminiImageSize(request dto.ImageRequest) string {
	for _, key := range []string{"imageSize", "image_size", "resolution"} {
		if value := imageStringExtra(request, key); value != "" {
			return value
		}
	}
	switch strings.ToLower(strings.TrimSpace(request.Quality)) {
	case "hd", "high", "2k":
		return "2K"
	case "4k":
		return "4K"
	case "standard", "medium", "low", "auto", "1k":
		return "1K"
	}
	return ""
}

func imageStringExtra(request dto.ImageRequest, key string) string {
	raw, ok := request.Extra[key]
	if !ok || len(raw) == 0 {
		return ""
	}
	var value string
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func guessGeminiImageMimeType(imageURL string) string {
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return "image/jpeg"
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

func GeminiNativeImageHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewOpenAIError(readErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var geminiResponse dto.GeminiChatResponse
	if jsonErr := common.Unmarshal(responseBody, &geminiResponse); jsonErr != nil {
		return nil, types.NewOpenAIError(jsonErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	openAIResponse := dto.ImageResponse{
		Created: common.GetTimestamp(),
		Data:    make([]dto.ImageData, 0),
	}
	for _, candidate := range geminiResponse.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil &&
				(part.InlineData.MimeType == "" || strings.HasPrefix(part.InlineData.MimeType, "image/")) &&
				part.InlineData.Data != "" {
				openAIResponse.Data = append(openAIResponse.Data, dto.ImageData{B64Json: part.InlineData.Data})
			}
			if part.FileData != nil &&
				(part.FileData.MimeType == "" || strings.HasPrefix(part.FileData.MimeType, "image/")) &&
				part.FileData.FileUri != "" {
				openAIResponse.Data = append(openAIResponse.Data, dto.ImageData{Url: part.FileData.FileUri})
			}
		}
	}
	if len(openAIResponse.Data) == 0 {
		return nil, types.NewOpenAIError(fmt.Errorf("no images generated"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	jsonResponse, jsonErr := common.Marshal(openAIResponse)
	if jsonErr != nil {
		return nil, types.NewError(jsonErr, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)

	usage := &dto.Usage{
		PromptTokens:     geminiResponse.UsageMetadata.PromptTokenCount,
		CompletionTokens: geminiResponse.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      geminiResponse.UsageMetadata.TotalTokenCount,
	}
	if usage.TotalTokens == 0 {
		const imageTokens = 258
		generatedImages := len(openAIResponse.Data)
		usage.PromptTokens = imageTokens * generatedImages
		usage.TotalTokens = imageTokens * generatedImages
	}
	if len(openAIResponse.Data) > 0 {
		info.PriceData.AddOtherRatio("n", float64(len(openAIResponse.Data)))
	}
	return usage, nil
}
