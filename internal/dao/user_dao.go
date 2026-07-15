package dao

import (
	"hotel-booking-go/internal/model"

	"gorm.io/gorm"
)

type UserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{db: db}
}

func (d *UserDAO) Create(user *model.User) error {
	return d.db.Create(user).Error
}

func (d *UserDAO) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := d.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *UserDAO) FindByID(id int64) (*model.User, error) {
	var user model.User
	err := d.db.Preload("Hotel").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *UserDAO) FindByHotelID(hotelID int64) ([]model.User, error) {
	var users []model.User
	err := d.db.Where("hotel_id = ? AND role IN ?", hotelID, []string{"hotel_admin", "front_desk"}).
		Preload("Hotel").Find(&users).Error
	return users, err
}

func (d *UserDAO) Update(user *model.User) error {
	return d.db.Save(user).Error
}

func (d *UserDAO) Delete(id int64) error {
	return d.db.Delete(&model.User{}, id).Error
}

func (d *UserDAO) FindUserIDsByHotelID(hotelID int64) ([]int64, error) {
	var ids []int64
	err := d.db.Model(&model.User{}).Where("hotel_id = ?", hotelID).Pluck("id", &ids).Error
	return ids, err
}
