package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

func SendImageAsyncWebhook(task *model.Task, payload any) error {
	imageAsync := task.PrivateData.ImageAsync
	if imageAsync == nil || imageAsync.WebhookURL == "" {
		return nil
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	headers := map[string]string{
		"Content-Type":       "application/json",
		"X-NewAPI-Task-ID":   task.TaskID,
		"X-NewAPI-Timestamp": timestamp,
	}
	if secret := os.Getenv("IMAGE_ASYNC_WEBHOOK_SECRET"); secret != "" {
		headers["X-NewAPI-Signature"] = "sha256=" + imageAsyncWebhookSignature(secret, timestamp, body)
	}

	var resp *http.Response
	if system_setting.EnableWorker() {
		resp, err = DoWorkerRequest(&WorkerRequest{
			URL:     imageAsync.WebhookURL,
			Key:     system_setting.WorkerValidKey,
			Method:  http.MethodPost,
			Headers: headers,
			Body:    json.RawMessage(body),
		})
	} else {
		fetchSetting := system_setting.GetFetchSetting()
		if err := common.ValidateURLWithFetchSetting(imageAsync.WebhookURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
			return fmt.Errorf("request reject: %v", err)
		}
		req, reqErr := http.NewRequest(http.MethodPost, imageAsync.WebhookURL, bytes.NewReader(body))
		if reqErr != nil {
			return reqErr
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		baseClient := GetHttpClient()
		if baseClient == nil {
			baseClient = http.DefaultClient
		}
		client := *baseClient
		client.Timeout = imageAsyncWebhookTimeout()
		resp, err = client.Do(req)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func imageAsyncWebhookTimeout() time.Duration {
	seconds := int64(10)
	if raw := os.Getenv("IMAGE_ASYNC_WEBHOOK_TIMEOUT_SECONDS"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second
}

func imageAsyncWebhookSignature(secret string, timestamp string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp))
	h.Write([]byte("."))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
