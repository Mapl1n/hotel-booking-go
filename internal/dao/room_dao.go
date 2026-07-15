package dao

import (
	"hotel-booking-go/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoomDAO struct {
	db *gorm.DB
}

func NewRoomDAO(db *gorm.DB) *RoomDAO {
	return &RoomDAO{db: db}
}

func (d *RoomDAO) FindByID(id int64) (*model.Room, error) {
	var room model.Room
	err := d.db.Preload("RoomType").First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// FindByIDForUpdate 带行锁查询（用于并发控制）
func (d *RoomDAO) FindByIDForUpdate(tx *gorm.DB, id int64) (*model.Room, error) {
	var room model.Room
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("RoomType").
		Where("id = ? AND status = ?", id, "available").
		First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (d *RoomDAO) FindAvailableByHotel(hotelID int64, checkIn, checkOut time.Time, roomTypeID *int64) ([]model.Room, error) {
	query := d.db.Preload("RoomType").Where("hotel_id = ? AND status = ?", hotelID, "available")
	if roomTypeID != nil {
		query = query.Where("room_type_id = ?", *roomTypeID)
	}

	// 批量获取所有潜在可用的房间
	var rooms []model.Room
	if err := query.Find(&rooms).Error; err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return rooms, nil
	}

	// 提取所有房间 ID
	roomIDs := make([]int64, len(rooms))
	for i, r := range rooms {
		roomIDs[i] = r.ID
	}

	// ★ 单次查询获取所有被占用的房间 ID（替代 N+1）
	var occupiedRoomIDs []int64
	d.db.Model(&model.Order{}).
		Select("DISTINCT room_id").
		Where("room_id IN ? AND status IN ? AND check_in < ? AND check_out > ?",
			roomIDs, []string{"pending", "paid", "checked_in"}, checkOut, checkIn,
		).
		Pluck("room_id", &occupiedRoomIDs)

	// 构建占用集合
	occupied := make(map[int64]bool, len(occupiedRoomIDs))
	for _, id := range occupiedRoomIDs {
		occupied[id] = true
	}

	// 过滤空闲房间
	available := make([]model.Room, 0, len(rooms))
	for _, room := range rooms {
		if !occupied[room.ID] {
			available = append(available, room)
		}
	}

	return available, nil
}

func (d *RoomDAO) FindAllByHotel(hotelID int64) ([]model.Room, error) {
	var rooms []model.Room
	err := d.db.Preload("RoomType").Where("hotel_id = ?", hotelID).Order("floor ASC, room_number ASC").Find(&rooms).Error
	return rooms, err
}

func (d *RoomDAO) FindByHotelAndNumber(hotelID int64, roomNumber string) (*model.Room, error) {
	var room model.Room
	err := d.db.Where("hotel_id = ? AND room_number = ?", hotelID, roomNumber).First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (d *RoomDAO) Create(room *model.Room) error {
	return d.db.Create(room).Error
}

func (d *RoomDAO) Update(room *model.Room) error {
	return d.db.Save(room).Error
}

// UpdateWithTx 在事务中更新房间状态（用于并发安全的 check-in/check-out/cancel）
func (d *RoomDAO) UpdateWithTx(tx *gorm.DB, room *model.Room) error {
	return tx.Model(room).Select("status").Updates(map[string]interface{}{
		"status": room.Status,
	}).Error
}

func (d *RoomDAO) Delete(id int64) error {
	return d.db.Delete(&model.Room{}, id).Error
}

func (d *RoomDAO) CountByHotel(hotelID int64) (int64, error) {
	var count int64
	err := d.db.Model(&model.Room{}).Where("hotel_id = ?", hotelID).Count(&count).Error
	return count, err
}

func (d *RoomDAO) CountAvailableByHotel(hotelID int64) (int64, error) {
	var count int64
	err := d.db.Model(&model.Room{}).Where("hotel_id = ? AND status = ?", hotelID, "available").Count(&count).Error
	return count, err
}

// ── RoomType DAO ──

type RoomTypeDAO struct {
	db *gorm.DB
}

func NewRoomTypeDAO(db *gorm.DB) *RoomTypeDAO {
	return &RoomTypeDAO{db: db}
}

func (d *RoomTypeDAO) FindByHotel(hotelID int64, activeOnly bool) ([]model.RoomType, error) {
	query := d.db.Where("hotel_id = ?", hotelID)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var types []model.RoomType
	err := query.Order("price ASC").Find(&types).Error
	return types, err
}

func (d *RoomTypeDAO) FindByID(id int64) (*model.RoomType, error) {
	var rt model.RoomType
	err := d.db.First(&rt, id).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (d *RoomTypeDAO) Create(rt *model.RoomType) error {
	return d.db.Create(rt).Error
}

func (d *RoomTypeDAO) Update(rt *model.RoomType) error {
	return d.db.Save(rt).Error
}
