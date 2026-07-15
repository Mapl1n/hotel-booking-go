package dao

import (
	"time"

	"hotel-booking-go/internal/model"

	"gorm.io/gorm"
)

type OrderDAO struct {
	db *gorm.DB
}

func NewOrderDAO(db *gorm.DB) *OrderDAO {
	return &OrderDAO{db: db}
}

func (d *OrderDAO) Transaction(fn func(tx *gorm.DB) error) error {
	return d.db.Transaction(fn)
}

func (d *OrderDAO) Create(tx *gorm.DB, order *model.Order) error {
	if tx == nil {
		tx = d.db
	}
	return tx.Create(order).Error
}

func (d *OrderDAO) FindByID(id int64) (*model.Order, error) {
	var order model.Order
	err := d.db.Preload("Hotel").Preload("Room.RoomType").Preload("User").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (d *OrderDAO) CountTodayOrders(date string) (int64, error) {
	var count int64
	err := d.db.Model(&model.Order{}).Where("order_no LIKE ?", "ORD"+date+"%").Count(&count).Error
	return count, err
}

func (d *OrderDAO) CountDateConflict(tx *gorm.DB, roomID int64, checkIn, checkOut time.Time, excludeID ...int64) (int64, error) {
	if tx == nil {
		tx = d.db
	}
	query := tx.Model(&model.Order{}).Where(
		"room_id = ? AND status IN ? AND check_in < ? AND check_out > ?",
		roomID, []string{"pending", "paid", "checked_in"}, checkOut, checkIn,
	)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (d *OrderDAO) ListByUser(userID int64) ([]model.Order, error) {
	var orders []model.Order
	err := d.db.Preload("Hotel").Preload("Room.RoomType").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

func (d *OrderDAO) ListByHotel(hotelID int64, status string) ([]model.Order, error) {
	var orders []model.Order
	query := d.db.Preload("Hotel").Preload("Room.RoomType").
		Where("hotel_id = ?", hotelID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at DESC").Find(&orders).Error
	return orders, err
}

// ListAll 所有订单（超管用）
func (d *OrderDAO) ListAll(status string) ([]model.Order, error) {
	var orders []model.Order
	query := d.db.Preload("Hotel").Preload("Room.RoomType")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at DESC").Limit(500).Find(&orders).Error
	return orders, err
}

func (d *OrderDAO) Update(tx *gorm.DB, order *model.Order) error {
	if tx == nil {
		tx = d.db
	}
	return tx.Save(order).Error
}

// FindExpiredPending 查找超时未支付的pending订单
func (d *OrderDAO) FindExpiredPending(minutes int) ([]model.Order, error) {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var orders []model.Order
	err := d.db.Preload("Room").Where(
		"status = ? AND created_at < ?", "pending", cutoff,
	).Find(&orders).Error
	return orders, err
}

// CountByStatus 统计某酒店指定状态的订单数
func (d *OrderDAO) CountByStatus(hotelID int64, status string, dates ...string) (int64, error) {
	query := d.db.Model(&model.Order{}).Where("hotel_id = ? AND status = ?", hotelID, status)
	if len(dates) > 0 && dates[0] != "" {
		if len(dates) == 1 {
			query = query.Where("DATE(check_in) = ?", dates[0])
		} else {
			query = query.Where("DATE(check_in) BETWEEN ? AND ?", dates[0], dates[1])
		}
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

// FindByHotelAndDateRange 按酒店+日期范围查订单
func (d *OrderDAO) FindByHotelAndDateRange(hotelID int64, startDate, endDate string, status string) ([]model.Order, error) {
	query := d.db.Preload("Room.RoomType").Where("hotel_id = ?", hotelID)
	if startDate != "" {
		query = query.Where("check_in >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("check_in <= ?", endDate)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var orders []model.Order
	err := query.Order("check_in DESC").Find(&orders).Error
	return orders, err
}

// FindByMonth 按月份查订单（房态日历用）
func (d *OrderDAO) FindByMonth(hotelID int64, firstDay, lastDay time.Time) ([]model.Order, error) {
	var orders []model.Order
	err := d.db.Where(
		"hotel_id = ? AND status IN ? AND check_in <= ? AND check_out > ?",
		hotelID, []string{"pending", "paid", "checked_in"}, lastDay, firstDay,
	).Find(&orders).Error
	return orders, err
}
