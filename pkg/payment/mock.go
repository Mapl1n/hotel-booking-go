package payment

import (
	"fmt"
	"time"

	"hotel-booking-go/internal/model"
)

type MockProvider struct{}

func (p *MockProvider) Name() string        { return "mock" }
func (p *MockProvider) DisplayName() string { return "模拟支付" }

func (p *MockProvider) CreatePayment(order *model.Order, payment *model.Payment) (*Result, error) {
	now := time.Now()
	payment.Status = "success"
	payment.PaidAt = &now
	if payment.PaymentNo == "" {
		payment.PaymentNo = fmt.Sprintf("MOCK%s", now.Format("20060102150405"))
	}
	order.Status = "paid"
	return &Result{
		Success:   true,
		PaymentNo: payment.PaymentNo,
		Message:   "支付成功（模拟）",
	}, nil
}

func (p *MockProvider) VerifyCallback(data map[string]interface{}, headers map[string]string) bool {
	return true
}

func (p *MockProvider) ParseCallback(data map[string]interface{}) CallbackData {
	pn, _ := data["payment_no"].(string)
	amt, _ := data["amount"].(float64)
	return CallbackData{PaymentNo: pn, Amount: amt, Status: "success"}
}
