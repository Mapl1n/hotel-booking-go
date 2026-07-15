package service

import (
	"errors"
	"fmt"
	"time"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/model"
	"hotel-booking-go/pkg/payment"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaymentService struct {
	paymentDAO *dao.PaymentDAO
	orderDAO   *dao.OrderDAO
	db         *gorm.DB
	payMode    string
}

func NewPaymentService(paymentDAO *dao.PaymentDAO, orderDAO *dao.OrderDAO, db *gorm.DB, payMode string) *PaymentService {
	return &PaymentService{paymentDAO: paymentDAO, orderDAO: orderDAO, db: db, payMode: payMode}
}

func (s *PaymentService) getProvider(method string) payment.Provider {
	if s.payMode != "production" && method != "mock" {
		return &payment.MockProvider{}
	}
	switch method {
	case "wechat":
		return &payment.WechatProvider{}
	case "alipay":
		return &payment.AlipayProvider{}
	default:
		return &payment.MockProvider{}
	}
}

// Create 创建支付（事务内 + 行锁保护）
func (s *PaymentService) Create(orderID int64, method string) (*PaymentResult, error) {
	var result *PaymentResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// ★ 锁定订单行，防止并发重复支付
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&order, orderID).Error; err != nil {
			return errors.New("订单不存在")
		}
		if order.Status != "pending" {
			return errors.New("当前状态不可支付")
		}

		// ★ 在事务内检查是否已有成功支付（行锁保护下）
		var existing model.Payment
		err := tx.Where("order_id = ? AND status = ?", orderID, "success").
			First(&existing).Error
		if err == nil {
			result = &PaymentResult{
				Success:   true,
				PaymentID: existing.ID,
				PaymentNo: existing.PaymentNo,
				Message:   "该订单已支付",
			}
			return nil
		}

		provider := s.getProvider(method)

		paymentNo := fmt.Sprintf("PAY%s%06d",
			time.Now().Format("20060102150405"),
			time.Now().Nanosecond()%1000000,
		)

		paymentModel := &model.Payment{
			HotelID:       order.HotelID,
			OrderID:       orderID,
			PaymentNo:     paymentNo,
			Amount:        order.TotalPrice,
			PaymentMethod: method,
			Status:        "pending",
		}
		if err := tx.Create(paymentModel).Error; err != nil {
			return err
		}

		// 调用支付提供商
		payResult, err := provider.CreatePayment(&order, paymentModel)
		if err != nil {
			return err
		}

		// ★ 根据提供商实际结果更新状态，不硬编码
		if payResult.Success {
			paymentModel.Status = "success"
			now := time.Now()
			paymentModel.PaidAt = &now
			order.Status = "paid"
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
		}
		if payResult.PaymentNo != "" && payResult.PaymentNo != paymentModel.PaymentNo {
			paymentModel.PaymentNo = payResult.PaymentNo
		}
		if err := tx.Save(paymentModel).Error; err != nil {
			return err
		}

		result = &PaymentResult{
			Success:   payResult.Success,
			PaymentID: paymentModel.ID,
			QRCode:    payResult.QRCode,
			PayURL:    payResult.PayURL,
			PaymentNo: paymentModel.PaymentNo,
			Message:   payResult.Message,
		}
		return nil
	})

	return result, err
}

func (s *PaymentService) GetStatus(paymentID int64) (*model.Payment, error) {
	return s.paymentDAO.FindByID(paymentID)
}

type PaymentResult struct {
	Success   bool   `json:"success"`
	PaymentID int64  `json:"payment_id"`
	QRCode    string `json:"qr_code,omitempty"`
	PayURL    string `json:"pay_url,omitempty"`
	PaymentNo string `json:"payment_no,omitempty"`
	Message   string `json:"message"`
}
