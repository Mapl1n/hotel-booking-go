package dao

import (
	"hotel-booking-go/internal/model"

	"gorm.io/gorm"
)

type HotelDAO struct {
	db *gorm.DB
}

func NewHotelDAO(db *gorm.DB) *HotelDAO {
	return &HotelDAO{db: db}
}

func (d *HotelDAO) FindAll() ([]model.Hotel, error) {
	var hotels []model.Hotel
	err := d.db.Order("id ASC").Find(&hotels).Error
	return hotels, err
}

func (d *HotelDAO) FindByID(id int64) (*model.Hotel, error) {
	var hotel model.Hotel
	err := d.db.First(&hotel, id).Error
	if err != nil {
		return nil, err
	}
	return &hotel, nil
}

func (d *HotelDAO) Create(hotel *model.Hotel) error {
	return d.db.Create(hotel).Error
}

func (d *HotelDAO) Update(hotel *model.Hotel) error {
	return d.db.Save(hotel).Error
}
