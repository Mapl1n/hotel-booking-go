package payment

import (
	"fmt"

	"hotel-booking-go/internal/model"
)

type AlipayProvider struct{}

func (p *AlipayProvider) Name() string        { return "alipay" }
func (p *AlipayProvider) DisplayName() string { return "支付宝" }

func (p *AlipayProvider) CreatePayment(order *model.Order, payment *model.Payment) (*Result, error) {
	return &Result{
		Success:   false,
		Message:   "支付宝暂未接入，请使用模拟支付",
		PaymentNo: payment.PaymentNo,
	}, fmt.Errorf("支付宝未配置")
}

func (p *AlipayProvider) VerifyCallback(data map[string]interface{}, headers map[string]string) bool {
	return false
}

func (p *AlipayProvider) ParseCallback(data map[string]interface{}) CallbackData {
	return CallbackData{}
}
