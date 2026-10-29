package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXunhuFeeIncludedAmountAddsTwoPercent(t *testing.T) {
	testCases := []struct {
		name   string
		amount float64
		want   string
	}{
		{name: "ten yuan", amount: 10, want: "10.20"},
		{name: "fractional yuan rounds to cents", amount: 12.34, want: "12.59"},
		{name: "half cent rounds up", amount: 0.25, want: "0.26"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := xunhuFeeIncludedAmount(decimal.NewFromFloat(tc.amount))
			assert.Equal(t, tc.want, got.StringFixed(2))
		})
	}
}

func TestRequestAmountIncludesXunhuFeeOnlyForXunhu(t *testing.T) {
	oldDB := model.DB
	oldMainType := common.MainDatabaseType()
	oldLogType := common.LogDatabaseType()
	oldMaster := common.IsMasterNode
	oldSQLitePath := common.SQLitePath
	oldRedisEnabled := common.RedisEnabled
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldPrice := operation_setting.Price
	oldMinTopUp := operation_setting.MinTopUp
	oldAppID := operation_setting.GetPaymentSetting().XunhuAppId
	oldPayMethods := operation_setting.PayMethods
	oldCompliance := operation_setting.GetPaymentSetting().ComplianceConfirmed
	oldTermsVersion := operation_setting.GetPaymentSetting().ComplianceTermsVersion
	common.RedisEnabled = false
	common.IsMasterNode = false
	common.SQLitePath = t.TempDir() + "/xunhu-fee-preview.db"
	t.Setenv("SQL_DSN", "")
	t.Setenv("LOG_SQL_DSN", "")
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.Price = 1
	operation_setting.MinTopUp = 1
	operation_setting.GetPaymentSetting().XunhuAppId = "xunhu-app"
	operation_setting.PayMethods = []map[string]string{{
		"provider":   model.PaymentProviderXunhu,
		"type":       "wxpay",
		"xunhu_type": "WAP",
	}}
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Setenv(operation_setting.XunhuAppSecretEnv, "xunhu-secret")
	t.Cleanup(func() {
		model.DB = oldDB
		common.SetDatabaseTypes(oldMainType, oldLogType)
		common.IsMasterNode = oldMaster
		common.SQLitePath = oldSQLitePath
		common.RedisEnabled = oldRedisEnabled
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
		operation_setting.Price = oldPrice
		operation_setting.MinTopUp = oldMinTopUp
		operation_setting.GetPaymentSetting().XunhuAppId = oldAppID
		operation_setting.PayMethods = oldPayMethods
		operation_setting.GetPaymentSetting().ComplianceConfirmed = oldCompliance
		operation_setting.GetPaymentSetting().ComplianceTermsVersion = oldTermsVersion
	})

	require.NoError(t, model.InitDB())
	db := model.DB
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.Create(&model.User{
		Id:       42,
		Username: "xunhu_fee_preview",
		Quota:    100_000_000,
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name     string
		body     string
		wantBody string
	}{
		{
			name:     "regular payment has no Xunhu surcharge",
			body:     `{"amount":10}`,
			wantBody: `{"message":"success","data":"10.00"}`,
		},
		{
			name:     "Xunhu payment includes two percent",
			body:     `{"amount":10,"provider":"xunhu","payment_method":"wxpay"}`,
			wantBody: `{"message":"success","data":"10.20","base_amount":"10.00","fee_amount":"0.20","fee_percent":2}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set("id", 42)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/amount", strings.NewReader(tc.body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			RequestAmount(ctx)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.JSONEq(t, tc.wantBody, recorder.Body.String())
		})
	}
}
