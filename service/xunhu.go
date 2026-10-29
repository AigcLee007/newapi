package service

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	xunhuAPIEndpoint   = "https://api.xunhupay.com/payment/do.html"
	xunhuResponseLimit = 1 << 20
)

type XunhuClient struct {
	appID      string
	appSecret  string
	endpoint   string
	httpClient *http.Client
}

type XunhuPaymentOrder struct {
	TradeOrderID string
	TotalFee     string
	Title        string
	NotifyURL    string
	ReturnURL    string
	Type         string
	WapURL       string
	WapName      string
}

type XunhuPayment struct {
	URL       string
	QRCodeURL string
}

func NewXunhuClient(appID string, appSecret string) *XunhuClient {
	return &XunhuClient{
		appID:      appID,
		appSecret:  appSecret,
		endpoint:   xunhuAPIEndpoint,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (client *XunhuClient) CreatePayment(ctx context.Context, order XunhuPaymentOrder) (*XunhuPayment, error) {
	if client == nil || client.appID == "" || client.appSecret == "" {
		return nil, errors.New("Xunhu client is not configured")
	}
	if order.TradeOrderID == "" || order.TotalFee == "" || order.Title == "" || order.NotifyURL == "" {
		return nil, errors.New("Xunhu payment parameters are incomplete")
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate Xunhu nonce: %w", err)
	}
	params := map[string]string{
		"version":        "1.1",
		"appid":          client.appID,
		"trade_order_id": order.TradeOrderID,
		"total_fee":      order.TotalFee,
		"title":          order.Title,
		"time":           strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":     order.NotifyURL,
		"nonce_str":      hex.EncodeToString(nonce),
	}
	if order.ReturnURL != "" {
		params["return_url"] = order.ReturnURL
	}
	if order.Type != "" {
		params["type"] = order.Type
	}
	if order.WapURL != "" {
		params["wap_url"] = order.WapURL
	}
	if order.WapName != "" {
		params["wap_name"] = order.WapName
	}
	params["hash"] = XunhuSignature(params, client.appSecret)

	body, err := common.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal Xunhu payment request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create Xunhu payment request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	httpClient := client.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send Xunhu payment request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Xunhu payment request returned HTTP %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, xunhuResponseLimit+1))
	if err != nil {
		return nil, fmt.Errorf("read Xunhu payment response: %w", err)
	}
	if len(responseBody) > xunhuResponseLimit {
		return nil, errors.New("Xunhu payment response is too large")
	}
	var rawParams map[string]json.RawMessage
	if err := common.Unmarshal(responseBody, &rawParams); err != nil {
		return nil, fmt.Errorf("decode Xunhu payment response: %w", err)
	}
	responseParams := make(map[string]string, len(rawParams))
	for key, rawValue := range rawParams {
		var stringValue string
		if err := common.Unmarshal(rawValue, &stringValue); err == nil {
			responseParams[key] = stringValue
			continue
		}
		var numberValue json.Number
		if err := common.Unmarshal(rawValue, &numberValue); err == nil {
			responseParams[key] = numberValue.String()
			continue
		}
		var boolValue bool
		if err := common.Unmarshal(rawValue, &boolValue); err == nil {
			responseParams[key] = strconv.FormatBool(boolValue)
			continue
		}
		if strings.TrimSpace(string(rawValue)) == "null" {
			responseParams[key] = ""
			continue
		}
		return nil, fmt.Errorf("Xunhu response field %q is not scalar", key)
	}
	if !VerifyXunhuSignature(responseParams, client.appSecret) {
		return nil, errors.New("Xunhu payment response signature verification failed")
	}
	errCode, err := strconv.Atoi(responseParams["errcode"])
	if err != nil || errCode != 0 {
		message := strings.TrimSpace(responseParams["errmsg"])
		if message == "" {
			message = "provider rejected the request"
		}
		return nil, fmt.Errorf("Xunhu payment request failed: %s", message)
	}

	payment := &XunhuPayment{
		URL:       strings.TrimSpace(responseParams["url"]),
		QRCodeURL: strings.TrimSpace(responseParams["url_qrcode"]),
	}
	if payment.URL == "" && payment.QRCodeURL == "" {
		return nil, errors.New("Xunhu response contains no payment URL")
	}
	if payment.URL != "" {
		parsedURL, err := url.ParseRequestURI(payment.URL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" || parsedURL.User != nil {
			return nil, errors.New("Xunhu response contains an invalid payment URL")
		}
	}
	return payment, nil
}

func XunhuSignature(params map[string]string, secret string) string {
	if secret == "" {
		return ""
	}

	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "hash" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var payload strings.Builder
	for _, key := range keys {
		if payload.Len() > 0 {
			payload.WriteByte('&')
		}
		payload.WriteString(key)
		payload.WriteByte('=')
		payload.WriteString(params[key])
	}
	payload.WriteString(secret)

	sum := md5.Sum([]byte(payload.String()))
	return hex.EncodeToString(sum[:])
}

func VerifyXunhuSignature(params map[string]string, secret string) bool {
	provided := params["hash"]
	if secret == "" || provided == "" {
		return false
	}

	expected := XunhuSignature(params, secret)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
