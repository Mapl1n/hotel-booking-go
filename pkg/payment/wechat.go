package payment

import (
	"fmt"

	"hotel-booking-go/internal/model"
)

type WechatProvider struct{}

func (p *WechatProvider) Name() string        { return "wechat" }
func (p *WechatProvider) DisplayName() string { return "微信支付" }

func (p *WechatProvider) CreatePayment(order *model.Order, payment *model.Payment) (*Result, error) {
	return &Result{
		Success:   false,
		Message:   "微信支付暂未接入，请使用模拟支付",
		PaymentNo: payment.PaymentNo,
	}, fmt.Errorf("微信支付未配置")
}

func (p *WechatProvider) VerifyCallback(data map[string]interface{}, headers map[string]string) bool {
	return false
}

func (p *WechatProvider) ParseCallback(data map[string]interface{}) CallbackData {
	return CallbackData{}
}
