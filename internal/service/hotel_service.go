package service

import (
	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/model"
)

type HotelService struct {
	hotelDAO *dao.HotelDAO
}

func NewHotelService(hotelDAO *dao.HotelDAO) *HotelService {
	return &HotelService{hotelDAO: hotelDAO}
}

func (s *HotelService) List() ([]model.Hotel, error) {
	return s.hotelDAO.FindAll()
}

func (s *HotelService) Get(id int64) (*model.Hotel, error) {
	return s.hotelDAO.FindByID(id)
}

func (s *HotelService) Create(name, address, phone, description string) (*model.Hotel, error) {
	hotel := &model.Hotel{
		Name:        name,
		Address:     address,
		Phone:       phone,
		Description: description,
	}
	err := s.hotelDAO.Create(hotel)
	return hotel, err
}

func (s *HotelService) Update(id int64, name, address, phone, description *string) (*model.Hotel, error) {
	hotel, err := s.hotelDAO.FindByID(id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		hotel.Name = *name
	}
	if address != nil {
		hotel.Address = *address
	}
	if phone != nil {
		hotel.Phone = *phone
	}
	if description != nil {
		hotel.Description = *description
	}
	err = s.hotelDAO.Update(hotel)
	return hotel, err
}
