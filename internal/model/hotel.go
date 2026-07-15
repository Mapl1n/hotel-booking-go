package model

import "time"

type Hotel struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;comment:酒店名称" json:"name"`
	Address     string    `gorm:"type:varchar(255);comment:酒店地址" json:"address"`
	Phone       string    `gorm:"type:varchar(20);comment:酒店联系电话" json:"phone"`
	Description string    `gorm:"type:varchar(500);comment:酒店简介" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	Users     []User     `gorm:"foreignKey:HotelID" json:"-"`
	RoomTypes []RoomType `gorm:"foreignKey:HotelID" json:"-"`
	Rooms     []Room     `gorm:"foreignKey:HotelID" json:"-"`
	Orders    []Order    `gorm:"foreignKey:HotelID" json:"-"`
}
