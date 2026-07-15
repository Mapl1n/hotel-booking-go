package service

import (
	"fmt"
	"time"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/model"
)

type AdminService struct {
	orderDAO   *dao.OrderDAO
	paymentDAO *dao.PaymentDAO
	roomDAO    *dao.RoomDAO
	hotelDAO   *dao.HotelDAO
	userDAO    *dao.UserDAO
}

func NewAdminService(orderDAO *dao.OrderDAO, paymentDAO *dao.PaymentDAO, roomDAO *dao.RoomDAO, hotelDAO *dao.HotelDAO, userDAO *dao.UserDAO) *AdminService {
	return &AdminService{orderDAO: orderDAO, paymentDAO: paymentDAO, roomDAO: roomDAO, hotelDAO: hotelDAO, userDAO: userDAO}
}

type DashboardStats struct {
	HotelName       string  `json:"hotel_name"`
	TodayCheckIns   int64   `json:"today_check_ins"`
	TodayCheckOuts  int64   `json:"today_check_outs"`
	CurrentOccupied int64   `json:"current_occupied"`
	TodayRevenue    float64 `json:"today_revenue"`
	OccupancyRate   string  `json:"occupancy_rate"`
	TotalRooms      int64   `json:"total_rooms"`
	AvailableRooms  int64   `json:"available_rooms"`
	WeekCheckIns    int64   `json:"week_check_ins,omitempty"`
	WeekRevenue     float64 `json:"week_revenue,omitempty"`
	MonthCheckIns   int64   `json:"month_check_ins,omitempty"`
	MonthRevenue    float64 `json:"month_revenue,omitempty"`
}

func (s *AdminService) GetDashboard(hotelID int64) (*DashboardStats, error) {
	hotel, err := s.hotelDAO.FindByID(hotelID)
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday())).Format("2006-01-02")
	monthStart := time.Now().Format("2006-01") + "-01"

	totalRooms, _ := s.roomDAO.CountByHotel(hotelID)
	availableRooms, _ := s.roomDAO.CountAvailableByHotel(hotelID)
	todayCheckIns, _ := s.orderDAO.CountByStatus(hotelID, "checked_in", today)
	todayCheckOuts, _ := s.orderDAO.CountByStatus(hotelID, "checked_out", today)
	todayRevenue, _ := s.paymentDAO.SumRevenueByDateRange(hotelID, today, today)
	weekCheckIns, _ := s.orderDAO.CountByStatus(hotelID, "checked_in", weekStart, today)
	weekRevenue, _ := s.paymentDAO.SumRevenueByDateRange(hotelID, weekStart, today)
	monthCheckIns, _ := s.orderDAO.CountByStatus(hotelID, "checked_in", monthStart, today)
	monthRevenue, _ := s.paymentDAO.SumRevenueByDateRange(hotelID, monthStart, today)

	rate := "0%"
	if totalRooms > 0 {
		rate = fmt.Sprintf("%.0f%%", float64(todayCheckIns)/float64(totalRooms)*100)
	}

	return &DashboardStats{
		HotelName:       hotel.Name,
		TodayCheckIns:   todayCheckIns,
		TodayCheckOuts:  todayCheckOuts,
		CurrentOccupied: todayCheckIns,
		TodayRevenue:    todayRevenue,
		OccupancyRate:   rate,
		TotalRooms:      totalRooms,
		AvailableRooms:  availableRooms,
		WeekCheckIns:    weekCheckIns,
		WeekRevenue:     weekRevenue,
		MonthCheckIns:   monthCheckIns,
		MonthRevenue:    monthRevenue,
	}, nil
}

func (s *AdminService) ToggleStaffActive(userID int64) (*model.User, error) {
	user, err := s.userDAO.FindByID(userID)
	if err != nil {
		return nil, err
	}
	user.IsActive = !user.IsActive
	err = s.userDAO.Update(user)
	return user, err
}

func (s *AdminService) DeleteStaff(userID int64) error {
	return s.userDAO.Delete(userID)
}
