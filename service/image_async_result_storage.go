package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	imageAsyncResultStoragePassthrough = "passthrough"
	imageAsyncResultStorageS3          = "s3"
	imageAsyncResultSourceB64          = "b64_json"
	imageAsyncResultSourceUpstreamURL  = "upstream_url"
)

var imageAsyncObjectPrefixPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9/_\-.]*$`)
var imageAsyncResultTaskIDReplacer = strings.NewReplacer("/", "_", "\\", "_", ".", "_", " ", "_")

type ImageAsyncObjectUploader interface {
	PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
}

type ImageAsyncObjectDeleter interface {
	DeleteObject(ctx context.Context, key string) error
}

type ImageAsyncResultDownloader interface {
	Download(ctx context.Context, sourceURL string, maxBytes int64) (*ImageAsyncDownloadedObject, error)
}

type ImageAsyncDownloadedObject struct {
	Body        []byte
	ContentType string
	Ext         string
	Size        int64
}

type imageAsyncResultStorageConfig struct {
	Mode             string
	Provider         string
	Endpoint         string
	Region           string
	Bucket           string
	AccessKeyID      string
	SecretAccessKey  string
	PublicBaseURL    string
	ForcePathStyle   bool
	ObjectPrefix     string
	ConvertB64ToURL  bool
	CacheUpstreamURL bool
	MaxBytes         int64
}

var (
	imageAsyncResultStorageMu          sync.Mutex
	imageAsyncResultUploaderOverride   ImageAsyncObjectUploader
	imageAsyncResultDownloaderOverride ImageAsyncResultDownloader
	imageAsyncResultNow                = time.Now
)

func NormalizeImageAsyncResultStorage(ctx context.Context, task *model.Task, responseBody []byte) ([]byte, error) {
	cfg, err := getImageAsyncResultStorageConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Mode != imageAsyncResultStorageS3 {
		return responseBody, nil
	}

	var response map[string]json.RawMessage
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("parse image async result: %w", err)
	}
	dataRaw, ok := response["data"]
	if !ok || len(bytes.TrimSpace(dataRaw)) == 0 {
		return responseBody, nil
	}

	var items []map[string]json.RawMessage
	if err := common.Unmarshal(dataRaw, &items); err != nil {
		return responseBody, nil
	}

	uploader, err := getImageAsyncObjectUploader(cfg)
	if err != nil {
		return nil, err
	}
	downloader := getImageAsyncResultDownloader()

	var uploaded []model.ImageAsyncResultObject
	committed := false
	defer func() {
		if committed || len(uploaded) == 0 {
			return
		}
		cleanupImageAsyncUploadedObjects(ctx, uploader, uploaded)
	}()

	for i := range items {
		item := items[i]
		if b64 := jsonStringValue(item["b64_json"]); b64 != "" && cfg.ConvertB64ToURL {
			body, contentType, ext, err := decodeImageAsyncB64JSON(b64, cfg.MaxBytes)
			if err != nil {
				return nil, err
			}
			obj, err := uploadImageAsyncResultObject(ctx, task, cfg, uploader, body, contentType, ext, i, imageAsyncResultSourceB64)
			if err != nil {
				return nil, err
			}
			uploaded = append(uploaded, obj)
			item["url"] = jsonStringRaw(obj.URL)
			item["b64_json"] = jsonStringRaw("")
			continue
		}

		if upstreamURL := jsonStringValue(item["url"]); upstreamURL != "" && cfg.CacheUpstreamURL {
			downloaded, err := downloader.Download(ctx, upstreamURL, cfg.MaxBytes)
			if err != nil {
				return nil, err
			}
			obj, err := uploadImageAsyncResultObject(ctx, task, cfg, uploader, downloaded.Body, downloaded.ContentType, downloaded.Ext, i, imageAsyncResultSourceUpstreamURL)
			if err != nil {
				return nil, err
			}
			uploaded = append(uploaded, obj)
			item["url"] = jsonStringRaw(obj.URL)
		}
	}

	if len(uploaded) == 0 {
		return responseBody, nil
	}

	data, err := common.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal image async result data: %w", err)
	}
	response["data"] = json.RawMessage(data)
	normalized, err := common.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("marshal image async result: %w", err)
	}
	if task.PrivateData.ImageAsync != nil {
		task.PrivateData.ImageAsync.ResultObjects = append(task.PrivateData.ImageAsync.ResultObjects, uploaded...)
	}
	committed = true
	return normalized, nil
}

func CleanupImageAsyncResultObjects(ctx context.Context, task *model.Task) {
	if task == nil || task.PrivateData.ImageAsync == nil || len(task.PrivateData.ImageAsync.ResultObjects) == 0 {
		return
	}
	cfg, err := getImageAsyncResultStorageConfig()
	if err != nil || cfg.Mode != imageAsyncResultStorageS3 {
		return
	}
	uploader, err := getImageAsyncObjectUploader(cfg)
	if err != nil {
		return
	}
	cleanupImageAsyncUploadedObjects(ctx, uploader, task.PrivateData.ImageAsync.ResultObjects)
}

func cleanupImageAsyncUploadedObjects(ctx context.Context, uploader ImageAsyncObjectUploader, objects []model.ImageAsyncResultObject) {
	deleter, ok := uploader.(ImageAsyncObjectDeleter)
	if !ok {
		return
	}
	for _, obj := range objects {
		if obj.ObjectKey == "" {
			continue
		}
		_ = deleter.DeleteObject(ctx, obj.ObjectKey)
	}
}

func getImageAsyncResultStorageConfig() (imageAsyncResultStorageConfig, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("IMAGE_ASYNC_RESULT_STORAGE")))
	if mode == "" {
		mode = imageAsyncResultStoragePassthrough
	}
	cfg := imageAsyncResultStorageConfig{
		Mode:             mode,
		Provider:         strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_PROVIDER")),
		Endpoint:         strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_ENDPOINT")),
		Region:           strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_REGION")),
		Bucket:           strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_BUCKET")),
		AccessKeyID:      strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_ACCESS_KEY_ID")),
		SecretAccessKey:  strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_SECRET_ACCESS_KEY")),
		PublicBaseURL:    strings.TrimRight(strings.TrimSpace(os.Getenv("IMAGE_ASYNC_S3_PUBLIC_BASE_URL")), "/"),
		ForcePathStyle:   parseImageAsyncBoolEnv("IMAGE_ASYNC_S3_FORCE_PATH_STYLE", false),
		ConvertB64ToURL:  parseImageAsyncBoolEnv("IMAGE_ASYNC_CONVERT_B64_TO_URL", true),
		CacheUpstreamURL: parseImageAsyncBoolEnv("IMAGE_ASYNC_CACHE_UPSTREAM_URL", true),
		MaxBytes:         getImageAsyncResultMaxBytes(),
	}
	if cfg.Provider == "" {
		cfg.Provider = "s3"
	}
	switch cfg.Mode {
	case imageAsyncResultStoragePassthrough:
		return cfg, nil
	case imageAsyncResultStorageS3:
		prefix, err := normalizeImageAsyncObjectPrefix(os.Getenv("IMAGE_ASYNC_S3_OBJECT_PREFIX"))
		if err != nil {
			return cfg, err
		}
		cfg.ObjectPrefix = prefix
		return cfg, nil
	default:
		return cfg, fmt.Errorf("unsupported IMAGE_ASYNC_RESULT_STORAGE: %s", cfg.Mode)
	}
}

func getImageAsyncObjectUploader(cfg imageAsyncResultStorageConfig) (ImageAsyncObjectUploader, error) {
	imageAsyncResultStorageMu.Lock()
	override := imageAsyncResultUploaderOverride
	imageAsyncResultStorageMu.Unlock()
	if override != nil {
		if cfg.PublicBaseURL == "" {
			return nil, fmt.Errorf("IMAGE_ASYNC_S3_PUBLIC_BASE_URL is required")
		}
		if cfg.Bucket == "" {
			cfg.Bucket = "mock"
		}
		return override, nil
	}
	return newImageAsyncS3Uploader(cfg)
}

func getImageAsyncResultDownloader() ImageAsyncResultDownloader {
	imageAsyncResultStorageMu.Lock()
	defer imageAsyncResultStorageMu.Unlock()
	if imageAsyncResultDownloaderOverride != nil {
		return imageAsyncResultDownloaderOverride
	}
	return imageAsyncHTTPDownloader{}
}

func parseImageAsyncBoolEnv(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getImageAsyncResultMaxBytes() int64 {
	mb := int64(64)
	if raw := os.Getenv("IMAGE_ASYNC_RESULT_MAX_MB"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			mb = parsed
		}
	}
	return mb << 20
}

func normalizeImageAsyncObjectPrefix(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" {
		return "", nil
	}
	raw = strings.Trim(raw, "/")
	if raw == "" {
		return "", nil
	}
	if strings.Contains(raw, "..") || path.IsAbs(raw) || !imageAsyncObjectPrefixPattern.MatchString(raw) {
		return "", fmt.Errorf("invalid IMAGE_ASYNC_S3_OBJECT_PREFIX")
	}
	return path.Clean(raw) + "/", nil
}

func jsonStringValue(raw json.RawMessage) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	var value string
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func jsonStringRaw(value string) json.RawMessage {
	data, _ := common.Marshal(value)
	return json.RawMessage(data)
}

func decodeImageAsyncB64JSON(value string, maxBytes int64) ([]byte, string, string, error) {
	if idx := strings.Index(value, ","); strings.HasPrefix(strings.ToLower(value[:max(0, idx)]), "data:image/") && idx >= 0 {
		value = value[idx+1:]
	}
	compact := strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(value)
	if compact == "" {
		return nil, "", "", fmt.Errorf("empty b64_json image result")
	}
	if rem := len(compact) % 4; rem != 0 {
		compact += strings.Repeat("=", 4-rem)
	}
	body, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		body, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(compact, "="))
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("decode b64_json image result: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", "", fmt.Errorf("image async result exceeds limit")
	}
	contentType, ext, err := detectImageAsyncContentType(body)
	if err != nil {
		return nil, "", "", err
	}
	return body, contentType, ext, nil
}

func detectImageAsyncContentType(body []byte) (string, string, error) {
	if len(body) >= 8 && bytes.Equal(body[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png", "png", nil
	}
	if len(body) >= 3 && body[0] == 0xff && body[1] == 0xd8 && body[2] == 0xff {
		return "image/jpeg", "jpg", nil
	}
	if len(body) >= 12 && bytes.Equal(body[:4], []byte("RIFF")) && bytes.Equal(body[8:12], []byte("WEBP")) {
		return "image/webp", "webp", nil
	}
	return "", "", fmt.Errorf("unsupported image async result image type")
}

func uploadImageAsyncResultObject(ctx context.Context, task *model.Task, cfg imageAsyncResultStorageConfig, uploader ImageAsyncObjectUploader, body []byte, contentType string, ext string, index int, source string) (model.ImageAsyncResultObject, error) {
	if int64(len(body)) > cfg.MaxBytes {
		return model.ImageAsyncResultObject{}, fmt.Errorf("image async result exceeds limit")
	}
	key := buildImageAsyncObjectKey(task.TaskID, cfg.ObjectPrefix, index, ext)
	if err := uploader.PutObject(ctx, key, bytes.NewReader(body), int64(len(body)), contentType); err != nil {
		return model.ImageAsyncResultObject{}, fmt.Errorf("upload image async result object: %w", err)
	}
	return model.ImageAsyncResultObject{
		Index:       index,
		Provider:    cfg.Provider,
		Bucket:      cfg.Bucket,
		ObjectKey:   key,
		URL:         joinImageAsyncPublicURL(cfg.PublicBaseURL, key),
		ContentType: contentType,
		Size:        int64(len(body)),
		Source:      source,
	}, nil
}

func buildImageAsyncObjectKey(taskID string, prefix string, index int, ext string) string {
	safeTaskID := sanitizeImageAsyncResultTaskID(taskID)
	if safeTaskID == "" {
		safeTaskID = "task"
	}
	now := imageAsyncResultNow()
	return fmt.Sprintf("%s%s/%s_%d.%s", prefix, now.Format("2006/01/02"), safeTaskID, index, ext)
}

func sanitizeImageAsyncResultTaskID(taskID string) string {
	taskID = imageAsyncResultTaskIDReplacer.Replace(strings.TrimSpace(taskID))
	var b strings.Builder
	for _, r := range taskID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func joinImageAsyncPublicURL(baseURL string, key string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(key, "/")
}

type imageAsyncHTTPDownloader struct{}

func (imageAsyncHTTPDownloader) Download(ctx context.Context, sourceURL string, maxBytes int64) (*ImageAsyncDownloadedObject, error) {
	if err := validateImageAsyncPublicURL(sourceURL); err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if err := validateImageAsyncPublicURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download image async result returned status %d", resp.StatusCode)
	}
	headerContentType := normalizeImageAsyncContentType(resp.Header.Get("Content-Type"))
	if headerContentType != "" && !strings.HasPrefix(headerContentType, "image/") {
		return nil, fmt.Errorf("download image async result is not an image")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("image async result exceeds limit")
	}
	contentType, ext, err := detectImageAsyncContentType(body)
	if err != nil {
		return nil, err
	}
	if headerContentType != "" && !imageAsyncContentTypesCompatible(headerContentType, contentType) {
		return nil, fmt.Errorf("download image async result content type mismatch")
	}
	return &ImageAsyncDownloadedObject{
		Body:        body,
		ContentType: contentType,
		Ext:         ext,
		Size:        int64(len(body)),
	}, nil
}

func validateImageAsyncPublicURL(sourceURL string) error {
	protection := &common.SSRFProtection{
		AllowPrivateIp:         false,
		DomainFilterMode:       false,
		IpFilterMode:           false,
		ApplyIPFilterForDomain: true,
	}
	if err := protection.ValidateURL(sourceURL); err != nil {
		return fmt.Errorf("request reject: %w", err)
	}
	return nil
}

func normalizeImageAsyncContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	if contentType == "image/jpg" {
		return "image/jpeg"
	}
	return contentType
}

func imageAsyncContentTypesCompatible(headerContentType string, detectedContentType string) bool {
	headerContentType = normalizeImageAsyncContentType(headerContentType)
	detectedContentType = normalizeImageAsyncContentType(detectedContentType)
	return headerContentType == "" || headerContentType == detectedContentType
}

type imageAsyncS3Uploader struct {
	cfg    imageAsyncResultStorageConfig
	client *http.Client
}

func newImageAsyncS3Uploader(cfg imageAsyncResultStorageConfig) (*imageAsyncS3Uploader, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("IMAGE_ASYNC_S3_ENDPOINT is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("IMAGE_ASYNC_S3_REGION is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("IMAGE_ASYNC_S3_BUCKET is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("IMAGE_ASYNC_S3_ACCESS_KEY_ID is required")
	}
	if cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("IMAGE_ASYNC_S3_SECRET_ACCESS_KEY is required")
	}
	if cfg.PublicBaseURL == "" {
		return nil, fmt.Errorf("IMAGE_ASYNC_S3_PUBLIC_BASE_URL is required")
	}
	if parsed, err := url.Parse(cfg.PublicBaseURL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid IMAGE_ASYNC_S3_PUBLIC_BASE_URL")
	}
	if parsed, err := url.Parse(cfg.Endpoint); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid IMAGE_ASYNC_S3_ENDPOINT")
	}
	return &imageAsyncS3Uploader{
		cfg: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (u *imageAsyncS3Uploader) PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	data, err := io.ReadAll(io.LimitReader(body, size+1))
	if err != nil {
		return err
	}
	if int64(len(data)) != size {
		return fmt.Errorf("image async result upload size mismatch")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.objectRequestURL(key), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Cache-Control", "public, max-age=31536000")
	return u.signAndDo(req, data, http.StatusOK)
}

func (u *imageAsyncS3Uploader) DeleteObject(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.objectRequestURL(key), nil)
	if err != nil {
		return err
	}
	return u.signAndDo(req, nil, http.StatusNoContent, http.StatusOK)
}

func (u *imageAsyncS3Uploader) signAndDo(req *http.Request, body []byte, successCodes ...int) error {
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	creds := aws.Credentials{
		AccessKeyID:     u.cfg.AccessKeyID,
		SecretAccessKey: u.cfg.SecretAccessKey,
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(req.Context(), creds, req, payloadHash, "s3", u.cfg.Region, time.Now()); err != nil {
		return err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	for _, code := range successCodes {
		if resp.StatusCode == code {
			return nil
		}
	}
	return fmt.Errorf("s3 object request failed with status %d", resp.StatusCode)
}

func (u *imageAsyncS3Uploader) objectRequestURL(key string) string {
	endpoint, _ := url.Parse(u.cfg.Endpoint)
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")
	escapedKey := escapeImageAsyncObjectKey(key)
	if u.cfg.ForcePathStyle {
		endpoint.Path = endpoint.Path + "/" + u.cfg.Bucket + "/" + escapedKey
		return endpoint.String()
	}
	endpoint.Host = u.cfg.Bucket + "." + endpoint.Host
	endpoint.Path = endpoint.Path + "/" + escapedKey
	return endpoint.String()
}

func escapeImageAsyncObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
