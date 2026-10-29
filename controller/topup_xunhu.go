package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const xunhuFeePercent int64 = 2

func xunhuFeeIncludedAmount(amount decimal.Decimal) decimal.Decimal {
	return amount.
		Mul(decimal.NewFromInt(100 + xunhuFeePercent)).
		Div(decimal.NewFromInt(100)).
		Round(2)
}

func RequestXunhuPay(c *gin.Context) {
	var req EpayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		paymentError(c, common.NewMessage("Invalid parameters"))
		return
	}
	if req.Amount < getMinTopup() {
		paymentError(c, common.NewMessage("Top-up amount cannot be less than {{min}}", map[string]any{"min": getMinTopup()}))
		return
	}

	userId := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, userId, req.Amount) {
		return
	}
	if !isXunhuTopUpEnabled() || !operation_setting.ContainsPayMethodForProvider(req.PaymentMethod, model.PaymentProviderXunhu) {
		paymentError(c, common.NewMessage("Payment method does not exist"))
		return
	}

	group, err := model.GetUserGroup(userId, true)
	if err != nil {
		paymentError(c, common.NewMessage("Failed to get the user group"))
		return
	}
	payAmount := xunhuFeeIncludedAmount(decimal.NewFromFloat(getPayMoney(req.Amount, group)).Round(2))
	if payAmount.LessThan(decimal.NewFromFloat(0.01)) {
		paymentError(c, common.NewMessage("Payment amount is too low"))
		return
	}

	var xunhuType string
	for _, method := range operation_setting.GetPayMethodsForProvider(model.PaymentProviderXunhu) {
		if method["type"] == req.PaymentMethod {
			xunhuType = strings.TrimSpace(method["xunhu_type"])
			break
		}
	}

	callbackAddress := service.GetCallbackAddress()
	callbackURL, err := url.ParseRequestURI(callbackAddress)
	if err != nil || (callbackURL.Scheme != "http" && callbackURL.Scheme != "https") || callbackURL.Host == "" || callbackURL.User != nil {
		paymentError(c, common.NewMessage("Payment information has not been configured by administrator"))
		return
	}
	notifyURL := callbackAddress + "/api/user/xunhu/notify"
	returnURL := paymentReturnPath("/wallet")
	tradeNo := fmt.Sprintf("USR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	storedAmount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		storedAmount = decimal.NewFromInt(req.Amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	topUp := &model.TopUp{
		UserId:          userId,
		Amount:          storedAmount,
		Money:           payAmount.InexactFloat64(),
		TradeNo:         tradeNo,
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: model.PaymentProviderXunhu,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), common.LogText("Xunhu failed to create top-up order user_id=%d trade_no=%s payment_method=%s amount=%d error=%q", userId, tradeNo, req.PaymentMethod, req.Amount, err.Error()))
		paymentError(c, common.NewMessage("Failed to create order"))
		return
	}

	paymentSetting := operation_setting.GetPaymentSetting()
	client := service.NewXunhuClient(paymentSetting.XunhuAppId, operation_setting.GetXunhuAppSecret())
	order := service.XunhuPaymentOrder{
		TradeOrderID: tradeNo,
		TotalFee:     payAmount.StringFixed(2),
		Title:        fmt.Sprintf("Wallet top-up %d", req.Amount),
		NotifyURL:    notifyURL,
		ReturnURL:    returnURL,
		Type:         xunhuType,
	}
	if xunhuType == "WAP" {
		order.WapURL = callbackURL.Scheme + "://" + callbackURL.Host
		order.WapName = strings.TrimSpace(common.SystemName)
		if order.WapName == "" {
			order.WapName = callbackURL.Hostname()
		}
	}
	payment, err := client.CreatePayment(c.Request.Context(), order)
	if err != nil {
		if statusErr := model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderXunhu, common.TopUpStatusExpired); statusErr != nil && !errors.Is(statusErr, model.ErrTopUpStatusInvalid) {
			logger.LogError(c.Request.Context(), common.LogText("Xunhu failed to expire incomplete top-up trade_no=%s error=%q", tradeNo, statusErr.Error()))
		}
		logger.LogError(c.Request.Context(), common.LogText("Xunhu failed to start payment user_id=%d trade_no=%s payment_method=%s error=%q", userId, tradeNo, req.PaymentMethod, err.Error()))
		paymentError(c, common.NewMessage("Failed to start payment"))
		return
	}
	logger.LogInfo(c.Request.Context(), common.LogText("Xunhu top-up order created user_id=%d trade_no=%s payment_method=%s amount=%d money=%s", userId, tradeNo, req.PaymentMethod, req.Amount, payAmount.StringFixed(2)))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"url":        payment.URL,
			"url_qrcode": payment.QRCodeURL,
		},
	})
}

func XunhuNotify(c *gin.Context) {
	if !isXunhuWebhookEnabled() || c.Request.Method != http.MethodPost {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	params := make(map[string]string, len(c.Request.PostForm))
	for key, values := range c.Request.PostForm {
		if len(values) != 1 {
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params[key] = values[0]
	}

	paymentSetting := operation_setting.GetPaymentSetting()
	appSecret := operation_setting.GetXunhuAppSecret()
	if len(params) == 0 || !service.VerifyXunhuSignature(params, appSecret) ||
		params["appid"] != paymentSetting.XunhuAppId || params["status"] != "OD" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	paidAmount, err := decimal.NewFromString(params["total_fee"])
	if err != nil || paidAmount.Sign() <= 0 || params["trade_order_id"] == "" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	tradeNo := params["trade_order_id"]
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	alreadyDone, err := model.RechargeXunhu(tradeNo, paidAmount, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTopUpNotFound), errors.Is(err, model.ErrPaymentMethodMismatch), errors.Is(err, model.ErrPaymentAmountMismatch), errors.Is(err, model.ErrTopUpStatusInvalid):
			logger.LogWarn(c.Request.Context(), common.LogText("Xunhu callback rejected trade_no=%s client_ip=%s reason=%q", tradeNo, c.ClientIP(), err.Error()))
		default:
			logger.LogError(c.Request.Context(), common.LogText("Xunhu top-up processing failed trade_no=%s client_ip=%s error=%q", tradeNo, c.ClientIP(), err.Error()))
		}
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if alreadyDone {
		logger.LogInfo(c.Request.Context(), common.LogText("Xunhu duplicate callback ignored trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
	}
	_, _ = c.Writer.Write([]byte("success"))
}
