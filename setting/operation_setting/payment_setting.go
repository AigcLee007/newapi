package operation_setting

import (
	"maps"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	AmountDiscount map[int]float64 `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠

	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
	XunhuAppId             string `json:"xunhu_app_id"`
}

const XunhuAppSecretEnv = "XUNHU_APP_SECRET"

const CurrentComplianceTermsVersion = "v1"

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:  []int{10, 20, 50, 100, 200, 500},
	AmountDiscount: map[int]float64{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

func IsPaymentComplianceConfirmed() bool {
	return paymentSetting.ComplianceConfirmed &&
		paymentSetting.ComplianceTermsVersion == CurrentComplianceTermsVersion
}

func GetXunhuAppSecret() string {
	return strings.TrimSpace(os.Getenv(XunhuAppSecretEnv))
}

func PaymentMethodProvider(method map[string]string) string {
	if provider := strings.ToLower(strings.TrimSpace(method["provider"])); provider != "" {
		return provider
	}
	switch method["type"] {
	case "stripe":
		return "stripe"
	case "waffo":
		return "waffo"
	case "waffo_pancake":
		return "waffo_pancake"
	default:
		return "epay"
	}
}

func GetPayMethodsForProvider(provider string) []map[string]string {
	methods := make([]map[string]string, 0)
	for _, method := range PayMethods {
		if PaymentMethodProvider(method) == provider {
			methods = append(methods, maps.Clone(method))
		}
	}
	return methods
}

func ContainsPayMethodForProvider(method string, provider string) bool {
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method && PaymentMethodProvider(payMethod) == provider {
			return true
		}
	}
	return false
}
