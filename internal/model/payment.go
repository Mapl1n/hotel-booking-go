package model

import "time"

type Payment struct {
	ID            int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	HotelID       int64      `gorm:"not null;comment:所属酒店" json:"hotel_id"`
	OrderID       int64      `gorm:"not null;index;comment:订单ID" json:"order_id"`
	PaymentNo     string     `gorm:"type:varchar(64);uniqueIndex;comment:支付流水号" json:"payment_no"`
	Amount        float64    `gorm:"not null;comment:支付金额" json:"amount"`
	PaymentMethod string     `gorm:"type:varchar(20);comment:支付方式" json:"payment_method"`
	Status        string     `gorm:"type:varchar(20);default:pending;comment:支付状态" json:"status"` // pending, success, failed
	PaidAt        *time.Time `gorm:"comment:支付时间" json:"paid_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`

	Order *Order `gorm:"foreignKey:OrderID" json:"-"`
}

// LoginLog 登录日志
type LoginLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;comment:用户ID" json:"user_id"`
	Username  string    `gorm:"type:varchar(50);not null;comment:登录账号" json:"username"`
	IPAddress string    `gorm:"type:varchar(50);not null;comment:登录IP" json:"ip_address"`
	LoginTime time.Time `gorm:"autoCreateTime;comment:登录时间" json:"login_time"`
}
