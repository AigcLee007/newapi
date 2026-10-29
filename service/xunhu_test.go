package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXunhuSignatureMatchesSortedParameterVector(t *testing.T) {
	params := map[string]string{
		"version":        "1.1",
		"trade_order_id": "USR1NO123",
		"empty":          "",
		"appid":          "app-id",
		"nonce_str":      "nonce-123",
		"total_fee":      "1.00",
		"hash":           "ignored",
	}

	assert.Equal(t, "6d8a0d864fc8f5b822555686ee1f467c", XunhuSignature(params, "secret-key"))
}

func TestVerifyXunhuSignatureRejectsMissingOrModifiedSignature(t *testing.T) {
	params := map[string]string{
		"appid":          "app-id",
		"nonce_str":      "nonce-123",
		"total_fee":      "1.00",
		"trade_order_id": "USR1NO123",
		"version":        "1.1",
	}
	params["hash"] = XunhuSignature(params, "secret-key")
	require.True(t, VerifyXunhuSignature(params, "secret-key"))

	params["total_fee"] = "1.01"
	assert.False(t, VerifyXunhuSignature(params, "secret-key"))

	delete(params, "hash")
	assert.False(t, VerifyXunhuSignature(params, "secret-key"))
}

func TestXunhuClientSendsSignedJSONAndVerifiesResponse(t *testing.T) {
	client := &XunhuClient{
		appID:     "app-id",
		appSecret: "secret-key",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		var submitted map[string]string
		require.NoError(t, common.Unmarshal(body, &submitted))
		assert.Equal(t, "app-id", submitted["appid"])
		assert.Equal(t, "USR1NO123", submitted["trade_order_id"])
		assert.Equal(t, "WAP", submitted["type"])
		assert.Equal(t, "https://dashboard.example.com", submitted["wap_url"])
		assert.Equal(t, "Gateway", submitted["wap_name"])
		assert.True(t, VerifyXunhuSignature(submitted, "secret-key"))

		responseParams := map[string]string{
			"errcode":    "0",
			"errmsg":     "success!",
			"url":        "https://pay.example/checkout",
			"url_qrcode": "weixin://pay/123",
		}
		responseParams["hash"] = XunhuSignature(responseParams, "secret-key")
		responseBody, err := common.Marshal(map[string]any{
			"errcode":    0,
			"errmsg":     responseParams["errmsg"],
			"url":        responseParams["url"],
			"url_qrcode": responseParams["url_qrcode"],
			"hash":       responseParams["hash"],
		})
		require.NoError(t, err)
		_, _ = writer.Write(responseBody)
	}))
	t.Cleanup(server.Close)
	client.endpoint = server.URL
	client.httpClient = server.Client()

	payment, err := client.CreatePayment(context.Background(), XunhuPaymentOrder{
		TradeOrderID: "USR1NO123",
		TotalFee:     "1.00",
		Title:        "Wallet top-up",
		NotifyURL:    "https://gateway.example/api/user/xunhu/notify",
		Type:         "WAP",
		WapURL:       "https://dashboard.example.com",
		WapName:      "Gateway",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://pay.example/checkout", payment.URL)
	assert.Equal(t, "weixin://pay/123", payment.QRCodeURL)
}

func TestXunhuClientRejectsInvalidResponseSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, `{"errcode":0,"url":"https://pay.example/checkout","hash":"invalid"}`)
	}))
	t.Cleanup(server.Close)
	client := &XunhuClient{
		appID:      "app-id",
		appSecret:  "secret-key",
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	_, err := client.CreatePayment(context.Background(), XunhuPaymentOrder{
		TradeOrderID: "USR1NO123",
		TotalFee:     "1.00",
		Title:        "Wallet top-up",
		NotifyURL:    "https://gateway.example/api/user/xunhu/notify",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "signature"))
}
