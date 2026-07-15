package model

import "time"

// RoomType 房型
type RoomType struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	HotelID     int64     `gorm:"not null;index;comment:所属酒店" json:"hotel_id"`
	Name        string    `gorm:"type:varchar(50);not null;comment:房型名称" json:"name"`
	Description string    `gorm:"type:varchar(255);comment:房型简介" json:"description"`
	Price       float64   `gorm:"not null;comment:每晚价格" json:"price"`
	Capacity    int       `gorm:"default:2;comment:可住人数" json:"capacity"`
	ImageURL    string    `gorm:"type:varchar(255)" json:"image_url"`
	IsActive    bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	Hotel *Hotel `gorm:"foreignKey:HotelID" json:"-"`
	Rooms []Room `gorm:"foreignKey:RoomTypeID" json:"-"`
}

// Room 房间
type Room struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	HotelID    int64     `gorm:"not null;uniqueIndex:idx_hotel_room;comment:所属酒店" json:"hotel_id"`
	RoomTypeID int64     `gorm:"not null;comment:房型ID" json:"room_type_id"`
	RoomNumber string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_hotel_room;comment:房间号" json:"room_number"`
	Floor      int       `gorm:"default:1;comment:楼层" json:"floor"`
	Status     string    `gorm:"type:varchar(20);default:available;comment:房间状态" json:"status"` // available, occupied, maintenance
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	Hotel    *Hotel    `gorm:"foreignKey:HotelID" json:"-"`
	RoomType *RoomType `gorm:"foreignKey:RoomTypeID" json:"room_type,omitempty"`
	Orders   []Order   `gorm:"foreignKey:RoomID" json:"-"`
}
