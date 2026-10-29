package controller

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func confirmPaymentComplianceForTest(t *testing.T) {
	t.Helper()
	paymentSetting := operation_setting.GetPaymentSetting()
	originalConfirmed := paymentSetting.ComplianceConfirmed
	originalTermsVersion := paymentSetting.ComplianceTermsVersion
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = originalConfirmed
		paymentSetting.ComplianceTermsVersion = originalTermsVersion
	})
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
}

func xunhuCallbackValues(params map[string]string) url.Values {
	values := make(url.Values, len(params))
	for key, value := range params {
		values.Set(key, value)
	}
	return values
}

func TestStripeWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPriceID := setting.StripePriceId
	t.Cleanup(func() {
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPriceID
	})

	setting.StripeWebhookSecret = ""
	setting.StripeApiSecret = "sk_test_123"
	setting.StripePriceId = "price_123"
	require.False(t, isStripeWebhookEnabled())

	setting.StripeWebhookSecret = "whsec_test"
	require.True(t, isStripeWebhookEnabled())

	setting.StripePriceId = ""
	require.False(t, isStripeWebhookEnabled())
}

func TestCreemWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalAPIKey := setting.CreemApiKey
	originalProducts := setting.CreemProducts
	originalWebhookSecret := setting.CreemWebhookSecret
	t.Cleanup(func() {
		setting.CreemApiKey = originalAPIKey
		setting.CreemProducts = originalProducts
		setting.CreemWebhookSecret = originalWebhookSecret
	})

	setting.CreemWebhookSecret = ""
	setting.CreemApiKey = "creem_api_key"
	setting.CreemProducts = `[{"productId":"prod_123"}]`
	require.False(t, isCreemWebhookEnabled())

	setting.CreemWebhookSecret = "creem_secret"
	require.True(t, isCreemWebhookEnabled())

	setting.CreemProducts = "[]"
	require.False(t, isCreemWebhookEnabled())
}

func TestWaffoWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalEnabled := setting.WaffoEnabled
	originalSandbox := setting.WaffoSandbox
	originalAPIKey := setting.WaffoApiKey
	originalPrivateKey := setting.WaffoPrivateKey
	originalPublicCert := setting.WaffoPublicCert
	originalSandboxAPIKey := setting.WaffoSandboxApiKey
	originalSandboxPrivateKey := setting.WaffoSandboxPrivateKey
	originalSandboxPublicCert := setting.WaffoSandboxPublicCert
	t.Cleanup(func() {
		setting.WaffoEnabled = originalEnabled
		setting.WaffoSandbox = originalSandbox
		setting.WaffoApiKey = originalAPIKey
		setting.WaffoPrivateKey = originalPrivateKey
		setting.WaffoPublicCert = originalPublicCert
		setting.WaffoSandboxApiKey = originalSandboxAPIKey
		setting.WaffoSandboxPrivateKey = originalSandboxPrivateKey
		setting.WaffoSandboxPublicCert = originalSandboxPublicCert
	})

	setting.WaffoEnabled = true
	setting.WaffoSandbox = false
	setting.WaffoApiKey = ""
	setting.WaffoPrivateKey = "private"
	setting.WaffoPublicCert = "public"
	require.False(t, isWaffoWebhookEnabled())

	setting.WaffoApiKey = "api"
	require.True(t, isWaffoWebhookEnabled())

	setting.WaffoEnabled = false
	require.False(t, isWaffoWebhookEnabled())

	setting.WaffoEnabled = true
	setting.WaffoSandbox = true
	setting.WaffoSandboxApiKey = ""
	setting.WaffoSandboxPrivateKey = "sandbox_private"
	setting.WaffoSandboxPublicCert = "sandbox_public"
	require.False(t, isWaffoWebhookEnabled())

	setting.WaffoSandboxApiKey = "sandbox_api"
	require.True(t, isWaffoWebhookEnabled())
}

func TestWaffoPancakeWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalMerchantID := setting.WaffoPancakeMerchantID
	originalPrivateKey := setting.WaffoPancakePrivateKey
	originalProductID := setting.WaffoPancakeProductID
	t.Cleanup(func() {
		setting.WaffoPancakeMerchantID = originalMerchantID
		setting.WaffoPancakePrivateKey = originalPrivateKey
		setting.WaffoPancakeProductID = originalProductID
	})

	// Presence of all three credentials enables the gateway. Webhook public
	// keys are bundled in the SDK and there is no separate Enabled toggle —
	// clear any of the three fields to disable.
	setting.WaffoPancakeMerchantID = ""
	setting.WaffoPancakePrivateKey = "private"
	setting.WaffoPancakeProductID = "product"
	require.False(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeMerchantID = "merchant"
	require.True(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeProductID = ""
	require.False(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeProductID = "product"
	setting.WaffoPancakePrivateKey = ""
	require.False(t, isWaffoPancakeWebhookEnabled())
}

func TestEpayWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})

	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "epay_id"
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}
	require.False(t, isEpayWebhookEnabled())

	operation_setting.EpayKey = "epay_key"
	require.True(t, isEpayWebhookEnabled())

	operation_setting.PayMethods = nil
	require.False(t, isEpayWebhookEnabled())
}

func TestXunhuTopUpRequiresCredentialsAndXunhuPaymentMethod(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	paymentSetting := operation_setting.GetPaymentSetting()
	originalAppID := paymentSetting.XunhuAppId
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		paymentSetting.XunhuAppId = originalAppID
		operation_setting.PayMethods = originalPayMethods
	})

	paymentSetting.XunhuAppId = "app-id"
	t.Setenv(operation_setting.XunhuAppSecretEnv, "app-secret")
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}
	require.False(t, isXunhuTopUpEnabled())

	operation_setting.PayMethods = []map[string]string{{"provider": "xunhu", "type": "WAP"}}
	paymentSetting.XunhuAppId = ""
	require.False(t, isXunhuTopUpEnabled())

	paymentSetting.XunhuAppId = "app-id"
	t.Setenv(operation_setting.XunhuAppSecretEnv, "")
	require.False(t, isXunhuTopUpEnabled())

	t.Setenv(operation_setting.XunhuAppSecretEnv, "app-secret")
	require.True(t, isXunhuTopUpEnabled())

	paymentSetting.ComplianceConfirmed = false
	require.False(t, isXunhuTopUpEnabled())
}

func TestXunhuMethodsDoNotEnableEpayAndAreReturnedWithProvider(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	paymentSetting := operation_setting.GetPaymentSetting()
	originalAppID := paymentSetting.XunhuAppId
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		paymentSetting.XunhuAppId = originalAppID
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})
	paymentSetting.XunhuAppId = "xunhu-app"
	t.Setenv(operation_setting.XunhuAppSecretEnv, "xunhu-secret")
	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{{
		"name":     "Xunhu Alipay",
		"type":     "alipay",
		"provider": model.PaymentProviderXunhu,
	}}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	GetTopUpInfo(ctx)
	var response struct {
		Data struct {
			EnableEpay  bool                `json:"enable_online_topup"`
			EnableXunhu bool                `json:"enable_xunhu_topup"`
			PayMethods  []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Data.EnableEpay)
	assert.True(t, response.Data.EnableXunhu)
	require.Len(t, response.Data.PayMethods, 1)
	assert.Equal(t, model.PaymentProviderXunhu, response.Data.PayMethods[0]["provider"])
}

func TestGetTopUpInfoHidesEpayMethodWhenOnlyXunhuIsConfigured(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	paymentSetting := operation_setting.GetPaymentSetting()
	originalAppID := paymentSetting.XunhuAppId
	originalPayMethods := operation_setting.PayMethods
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	t.Cleanup(func() {
		paymentSetting.XunhuAppId = originalAppID
		operation_setting.PayMethods = originalPayMethods
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
	})
	paymentSetting.XunhuAppId = "xunhu-app"
	t.Setenv(operation_setting.XunhuAppSecretEnv, "xunhu-secret")
	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{
		{"name": "Alipay", "type": "alipay"},
		{"name": "Xunhu Alipay", "type": "alipay", "provider": model.PaymentProviderXunhu},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)
	GetTopUpInfo(ctx)

	var response struct {
		Data struct {
			EnableOnlineTopUp bool                `json:"enable_online_topup"`
			EnableXunhuTopUp  bool                `json:"enable_xunhu_topup"`
			PayMethods        []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Data.EnableOnlineTopUp)
	assert.True(t, response.Data.EnableXunhuTopUp)
	assert.Len(t, response.Data.PayMethods, 1)
	assert.Equal(t, model.PaymentProviderXunhu, response.Data.PayMethods[0]["provider"])
}

func TestGetOptionsNeverReturnsXunhuAppSecret(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	originalOptions := maps.Clone(common.OptionMap)
	common.OptionMap = map[string]string{
		"payment_setting.xunhu_app_id": "visible-xunhu-app",
	}
	common.OptionMapRWMutex.Unlock()
	t.Setenv(operation_setting.XunhuAppSecretEnv, "private-xunhu-secret")
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	GetOptions(ctx)

	var response struct {
		Data []model.Option `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	values := make(map[string]string, len(response.Data))
	for _, option := range response.Data {
		values[option.Key] = option.Value
	}
	assert.Equal(t, "visible-xunhu-app", values["payment_setting.xunhu_app_id"])
	assert.Equal(t, "true", values["payment_setting.xunhu_app_secret_configured"])
	assert.NotContains(t, values, "payment_setting.xunhu_app_secret")
	assert.NotContains(t, recorder.Body.String(), "private-xunhu-secret")
}

func TestEpayCallbacksVerifyBeforeSettlement(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalServerAddress := system_setting.ServerAddress
	originalDB := model.DB
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		system_setting.ServerAddress = originalServerAddress
		model.DB = originalDB
	})
	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "epay_id"
	operation_setting.EpayKey = "test-key"
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}
	system_setting.ServerAddress = "https://dashboard.example.com"
	// Rejected callbacks and valid non-success events must never reach settlement.
	// A nil database makes an accidental attempt to settle fail the test.
	model.DB = nil
	require.True(t, isEpayWebhookEnabled())

	// Both ambiguous maps below had this same valid signing string in v0.0.4.
	// This must fail because of parameter ambiguity, not a mismatched signature.
	ambiguousSign := epay.MD5String("money=1.00&name=item&out_trade_no=other&out_trade_no=order&pid=epay_id&trade_status=TRADE_SUCCESS&type=alipay", "test-key")
	pendingSign := epay.MD5String("money=1.00&name=item%26other&out_trade_no=order&pid=epay_id&trade_status=WAIT_BUYER_PAY&type=alipay", "test-key")
	cases := []struct {
		name         string
		productName  string
		tradeNo      string
		status       string
		sign         string
		validPending bool
	}{
		{"separator in product name", "item&out_trade_no=other", "order", epay.StatusTradeSuccess, ambiguousSign, false},
		{"separator in order number", "item", "other&out_trade_no=order", epay.StatusTradeSuccess, ambiguousSign, false},
		{"invalid signature", "item", "order", epay.StatusTradeSuccess, "invalid", false},
		{"missing signature", "item", "order", epay.StatusTradeSuccess, "", false},
		{"valid pending with literal percent escape", "item%26other", "order", "WAIT_BUYER_PAY", pendingSign, true},
	}
	endpoints := []struct {
		name             string
		path             string
		handler          gin.HandlerFunc
		status           int
		rejectedBody     string
		pendingBody      string
		rejectedLocation string
		pendingLocation  string
	}{
		{
			name: "topup notify", path: "/api/user/epay/notify", handler: EpayNotify,
			status: http.StatusOK, rejectedBody: "fail", pendingBody: "success",
		},
		{
			name: "subscription notify", path: "/api/subscription/epay/notify", handler: SubscriptionEpayNotify,
			status: http.StatusOK, rejectedBody: "fail", pendingBody: "fail",
		},
		{
			name: "subscription return", path: "/api/subscription/epay/return", handler: SubscriptionEpayReturn,
			status:           http.StatusFound,
			rejectedLocation: "https://dashboard.example.com/wallet?pay=fail",
			pendingLocation:  "https://dashboard.example.com/wallet?pay=pending",
		},
	}
	router := gin.New()
	for _, endpoint := range endpoints {
		router.Any(endpoint.path, endpoint.handler)
	}
	for _, endpoint := range endpoints {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			for _, tc := range cases {
				t.Run(endpoint.name+"/"+method+"/"+tc.name, func(t *testing.T) {
					params := url.Values{
						"pid": {"epay_id"}, "type": {"alipay"}, "money": {"1.00"},
						"name": {tc.productName}, "out_trade_no": {tc.tradeNo},
						"trade_status": {tc.status}, "sign_type": {"MD5"},
					}
					if tc.sign != "" {
						params.Set("sign", tc.sign)
					}
					request := httptest.NewRequest(method, endpoint.path+"?"+params.Encode(), nil)
					if method == http.MethodPost {
						request = httptest.NewRequest(method, endpoint.path, strings.NewReader(params.Encode()))
						request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					}
					recorder := httptest.NewRecorder()
					require.NotPanics(t, func() { router.ServeHTTP(recorder, request) })
					assert.Equal(t, endpoint.status, recorder.Code)
					wantBody, wantLocation := endpoint.rejectedBody, endpoint.rejectedLocation
					if tc.validPending {
						wantBody, wantLocation = endpoint.pendingBody, endpoint.pendingLocation
					}
					if wantBody != "" {
						assert.Equal(t, wantBody, recorder.Body.String())
					}
					assert.Equal(t, wantLocation, recorder.Header().Get("Location"))
				})
			}
		}
	}
}

func TestXunhuNotifyValidatesPaymentBeforeSettlement(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	paymentSetting := operation_setting.GetPaymentSetting()
	originalAppID := paymentSetting.XunhuAppId

	originalPayMethods := operation_setting.PayMethods
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMainType, originalLogType := common.MainDatabaseType(), common.LogDatabaseType()
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		paymentSetting.XunhuAppId = originalAppID
		operation_setting.PayMethods = originalPayMethods
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainType, originalLogType)
		common.RedisEnabled = originalRedisEnabled
	})
	paymentSetting.XunhuAppId = "xunhu-app"
	t.Setenv(operation_setting.XunhuAppSecretEnv, "xunhu-secret")
	operation_setting.PayMethods = []map[string]string{{"provider": "xunhu", "type": "alipay"}}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.Log{}))
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.Create(&model.User{
		Id:       901,
		Username: "xunhu_callback_user",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, (&model.TopUp{
		UserId:          901,
		Amount:          2,
		Money:           1.50,
		TradeNo:         "xunhu-callback-order",
		PaymentMethod:   "alipay",
		PaymentProvider: model.PaymentProviderXunhu,
		Status:          common.TopUpStatusPending,
	}).Insert())

	validParams := map[string]string{
		"appid":          "xunhu-app",
		"trade_order_id": "xunhu-callback-order",
		"total_fee":      "1.50",
		"transaction_id": "provider-transaction",
		"status":         "OD",
		"time":           "1720000000",
		"nonce_str":      "callback-nonce",
	}
	validParams["hash"] = service.XunhuSignature(validParams, operation_setting.GetXunhuAppSecret())

	cases := []struct {
		name       string
		mutate     func(map[string]string)
		wantResult string
	}{
		{name: "valid signed success", mutate: func(map[string]string) {}, wantResult: "success"},
		{name: "missing signature", mutate: func(params map[string]string) { delete(params, "hash") }, wantResult: "fail"},
		{name: "invalid signature", mutate: func(params map[string]string) { params["hash"] = "invalid" }, wantResult: "fail"},
		{name: "wrong app id", mutate: func(params map[string]string) { params["appid"] = "other-app" }, wantResult: "fail"},
		{name: "unpaid or refunded order", mutate: func(params map[string]string) { params["status"] = "CD" }, wantResult: "fail"},
		{name: "wrong amount", mutate: func(params map[string]string) { params["total_fee"] = "1.49" }, wantResult: "fail"},
		{name: "unknown local order", mutate: func(params map[string]string) { params["trade_order_id"] = "unknown-order" }, wantResult: "fail"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := maps.Clone(validParams)
			tc.mutate(params)
			if tc.name != "missing signature" && tc.name != "invalid signature" {
				params["hash"] = service.XunhuSignature(params, operation_setting.GetXunhuAppSecret())
			}
			request := httptest.NewRequest(http.MethodPost, "/api/user/xunhu/notify", strings.NewReader(xunhuCallbackValues(params).Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request

			XunhuNotify(ctx)

			assert.Equal(t, tc.wantResult, recorder.Body.String())
		})
	}
	assert.Equal(t, common.TopUpStatusSuccess, model.GetTopUpByTradeNo("xunhu-callback-order").Status)
	assert.Equal(t, int(common.QuotaPerUnit*2), func() int {
		var user model.User
		require.NoError(t, db.Select("quota").First(&user, 901).Error)
		return user.Quota
	}())

	request := httptest.NewRequest(http.MethodPost, "/api/user/xunhu/notify", strings.NewReader(xunhuCallbackValues(validParams).Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	XunhuNotify(ctx)
	assert.Equal(t, "success", recorder.Body.String())
}

func TestXunhuNotifyRejectsDisabledGatewayAndInvalidAmount(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	paymentSetting := operation_setting.GetPaymentSetting()
	originalAppID := paymentSetting.XunhuAppId
	originalPayMethods := operation_setting.PayMethods
	originalDB := model.DB
	t.Cleanup(func() {
		paymentSetting.XunhuAppId = originalAppID
		operation_setting.PayMethods = originalPayMethods
		model.DB = originalDB
	})
	paymentSetting.XunhuAppId = "xunhu-app"
	t.Setenv(operation_setting.XunhuAppSecretEnv, "xunhu-secret")
	operation_setting.PayMethods = []map[string]string{{"provider": "xunhu", "type": "alipay"}}
	model.DB = nil

	params := map[string]string{
		"appid":          "xunhu-app",
		"trade_order_id": "any-order",
		"total_fee":      "not-money",
		"transaction_id": "provider-transaction",
		"status":         "OD",
		"hash":           "",
	}
	params["hash"] = service.XunhuSignature(params, operation_setting.GetXunhuAppSecret())
	request := httptest.NewRequest(http.MethodPost, "/api/user/xunhu/notify", strings.NewReader(xunhuCallbackValues(params).Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	XunhuNotify(ctx)
	assert.Equal(t, "fail", recorder.Body.String())

	operation_setting.PayMethods = nil
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = request
	XunhuNotify(ctx)
	assert.Equal(t, "fail", recorder.Body.String())
}
