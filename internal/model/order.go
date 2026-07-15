package model

import "time"

type Order struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	HotelID      int64      `gorm:"not null;index;comment:所属酒店" json:"hotel_id"`
	UserID       int64      `gorm:"not null;index;comment:用户ID" json:"user_id"`
	RoomID       int64      `gorm:"not null;index;comment:房间ID" json:"room_id"`
	OrderNo      string     `gorm:"type:varchar(20);uniqueIndex;comment:订单号" json:"order_no"`
	GuestName    string     `gorm:"type:varchar(50);not null;comment:入住人姓名" json:"guest_name"`
	IDCard       string     `gorm:"type:varchar(255);not null;comment:身份证号AES加密" json:"id_card"`
	CheckIn      time.Time  `gorm:"type:date;not null;comment:入住日期" json:"check_in"`
	CheckOut     time.Time  `gorm:"type:date;not null;comment:退房日期" json:"check_out"`
	TotalPrice   float64    `gorm:"not null;comment:总价" json:"total_price"`
	Status       string     `gorm:"type:varchar(20);default:pending;comment:订单状态" json:"status"` // pending, paid, checked_in, checked_out, cancelled
	CancelledAt  *time.Time `gorm:"comment:取消时间" json:"cancelled_at"`
	CancelReason string     `gorm:"type:varchar(200);comment:取消原因" json:"cancel_reason"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`

	Hotel    *Hotel    `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
	Room     *Room     `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	User     *User     `gorm:"foreignKey:UserID" json:"-"`
	Payments []Payment `gorm:"foreignKey:OrderID" json:"-"`
}
