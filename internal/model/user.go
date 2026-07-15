package model

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"type:varchar(50);uniqueIndex;not null;comment:手机号登录" json:"username"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Phone        string    `gorm:"type:varchar(20);comment:联系电话" json:"phone"`
	Role         string    `gorm:"type:varchar(20);default:guest;comment:用户角色" json:"role"` // super_admin, hotel_admin, front_desk, guest
	HotelID      *int64    `gorm:"comment:所属酒店（超管为NULL）" json:"hotel_id"`
	IsActive     bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	Hotel *Hotel `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
}
