package dao

import (
	"hotel-booking-go/internal/model"

	"gorm.io/gorm"
)

type PaymentDAO struct {
	db *gorm.DB
}

func NewPaymentDAO(db *gorm.DB) *PaymentDAO {
	return &PaymentDAO{db: db}
}

func (d *PaymentDAO) Create(tx *gorm.DB, payment *model.Payment) error {
	if tx == nil {
		tx = d.db
	}
	return tx.Create(payment).Error
}

func (d *PaymentDAO) FindByID(id int64) (*model.Payment, error) {
	var payment model.Payment
	err := d.db.First(&payment, id).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (d *PaymentDAO) FindByPaymentNo(paymentNo string) (*model.Payment, error) {
	var payment model.Payment
	err := d.db.Where("payment_no = ?", paymentNo).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (d *PaymentDAO) FindSuccessByOrder(orderID int64) (*model.Payment, error) {
	var payment model.Payment
	err := d.db.Where("order_id = ? AND status = ?", orderID, "success").First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (d *PaymentDAO) Update(tx *gorm.DB, payment *model.Payment) error {
	if tx == nil {
		tx = d.db
	}
	return tx.Save(payment).Error
}

// SumRevenueByDateRange 按日期范围统计已支付金额
func (d *PaymentDAO) SumRevenueByDateRange(hotelID int64, startDate, endDate string) (float64, error) {
	var total float64
	query := d.db.Model(&model.Payment{}).
		Where("hotel_id = ? AND status = ?", hotelID, "success")
	if startDate != "" && endDate != "" {
		query = query.Where("DATE(paid_at) BETWEEN ? AND ?", startDate, endDate)
	} else if startDate != "" {
		query = query.Where("DATE(paid_at) = ?", startDate)
	}
	err := query.Select("COALESCE(SUM(amount), 0)").Row().Scan(&total)
	return total, err
}

// ── LoginLog DAO ──

type LoginLogDAO struct {
	db *gorm.DB
}

func NewLoginLogDAO(db *gorm.DB) *LoginLogDAO {
	return &LoginLogDAO{db: db}
}

func (d *LoginLogDAO) Create(log *model.LoginLog) error {
	return d.db.Create(log).Error
}

func (d *LoginLogDAO) ListByUserIDs(userIDs []int64, limit int) ([]model.LoginLog, error) {
	var logs []model.LoginLog
	err := d.db.Where("user_id IN ?", userIDs).
		Order("login_time DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func (d *LoginLogDAO) ListAll(limit int) ([]model.LoginLog, error) {
	var logs []model.LoginLog
	err := d.db.Order("login_time DESC").Limit(limit).Find(&logs).Error
	return logs, err
}
