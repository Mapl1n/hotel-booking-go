package payment

import "hotel-booking-go/internal/model"

// Provider 支付提供商接口
type Provider interface {
	Name() string
	DisplayName() string
	CreatePayment(order *model.Order, payment *model.Payment) (*Result, error)
	VerifyCallback(data map[string]interface{}, headers map[string]string) bool
	ParseCallback(data map[string]interface{}) CallbackData
}

type Result struct {
	Success   bool   `json:"success"`
	QRCode    string `json:"qr_code,omitempty"`
	PayURL    string `json:"pay_url,omitempty"`
	PaymentNo string `json:"payment_no,omitempty"`
	Message   string `json:"message"`
}

type CallbackData struct {
	PaymentNo string `json:"payment_no"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"` // "success" or "failed"
}

// Methods 支付方式列表
var Methods = []map[string]string{
	{"value": "wechat", "label": "微信支付", "icon": "💚"},
	{"value": "alipay", "label": "支付宝", "icon": "💙"},
	{"value": "bankcard", "label": "银行卡", "icon": "💳"},
}
