package service

import (
	"errors"
	"fmt"
	"time"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/model"

	"github.com/xuri/excelize/v2"
)

type RoomService struct {
	roomDAO     *dao.RoomDAO
	roomTypeDAO *dao.RoomTypeDAO
	orderDAO    *dao.OrderDAO
	hotelDAO    *dao.HotelDAO
}

func NewRoomService(roomDAO *dao.RoomDAO, roomTypeDAO *dao.RoomTypeDAO, orderDAO *dao.OrderDAO, hotelDAO *dao.HotelDAO) *RoomService {
	return &RoomService{roomDAO: roomDAO, roomTypeDAO: roomTypeDAO, orderDAO: orderDAO, hotelDAO: hotelDAO}
}

var (
	ErrRoomNumberExists = errors.New("该房间号已存在")
	ErrRoomOccupied     = errors.New("已入住房间不能设为维护")
	ErrRoomInUse        = errors.New("仅空闲或维护中的房间可删除")
)

// ── 房型 ──

func (s *RoomService) ListRoomTypes(hotelID int64, activeOnly bool) ([]model.RoomType, error) {
	return s.roomTypeDAO.FindByHotel(hotelID, activeOnly)
}

func (s *RoomService) CreateRoomType(hotelID int64, name string, price float64, capacity int, description string) (*model.RoomType, error) {
	rt := &model.RoomType{
		HotelID:     hotelID,
		Name:        name,
		Price:       price,
		Capacity:    capacity,
		Description: description,
		IsActive:    true,
	}
	err := s.roomTypeDAO.Create(rt)
	return rt, err
}

func (s *RoomService) UpdateRoomType(id int64, name *string, price *float64, capacity *int, description *string, isActive *bool) (*model.RoomType, error) {
	rt, err := s.roomTypeDAO.FindByID(id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		rt.Name = *name
	}
	if price != nil {
		rt.Price = *price
	}
	if capacity != nil {
		rt.Capacity = *capacity
	}
	if description != nil {
		rt.Description = *description
	}
	if isActive != nil {
		rt.IsActive = *isActive
	}
	err = s.roomTypeDAO.Update(rt)
	return rt, err
}

// ── 房间 ──

func (s *RoomService) GetAvailableRooms(hotelID int64, checkIn, checkOut time.Time, roomTypeID *int64) ([]model.Room, error) {
	return s.roomDAO.FindAvailableByHotel(hotelID, checkIn, checkOut, roomTypeID)
}

func (s *RoomService) GetAllRooms(hotelID int64) ([]model.Room, error) {
	return s.roomDAO.FindAllByHotel(hotelID)
}

func (s *RoomService) CreateRoom(hotelID, roomTypeID int64, roomNumber string, floor int) (*model.Room, error) {
	// 验证房型属于该酒店
	rt, err := s.roomTypeDAO.FindByID(roomTypeID)
	if err != nil {
		return nil, errors.New("房型不存在")
	}
	if rt.HotelID != hotelID {
		return nil, errors.New("房型不属于该酒店")
	}
	// 检查房间号是否已存在
	_, err = s.roomDAO.FindByHotelAndNumber(hotelID, roomNumber)
	if err == nil {
		return nil, ErrRoomNumberExists
	}
	room := &model.Room{
		HotelID:    hotelID,
		RoomTypeID: roomTypeID,
		RoomNumber: roomNumber,
		Floor:      floor,
		Status:     "available",
	}
	err = s.roomDAO.Create(room)
	return room, err
}

func (s *RoomService) BatchCreateRooms(hotelID, roomTypeID int64, floor, startNumber, count int, prefix string) ([]string, error) {
	rt, err := s.roomTypeDAO.FindByID(roomTypeID)
	if err != nil {
		return nil, errors.New("房型不存在")
	}
	if rt.HotelID != hotelID {
		return nil, errors.New("房型不属于该酒店")
	}
	var created []string
	for i := 0; i < count; i++ {
		num := startNumber + i
		roomNumber := fmt.Sprintf("%s%02d", prefix, num)
		if prefix == "" {
			roomNumber = fmt.Sprintf("%d%02d", floor, num)
		}
		if _, err := s.roomDAO.FindByHotelAndNumber(hotelID, roomNumber); err == nil {
			continue // 跳过已存在的
		}
		room := &model.Room{
			HotelID:    hotelID,
			RoomTypeID: roomTypeID,
			RoomNumber: roomNumber,
			Floor:      floor,
			Status:     "available",
		}
		s.roomDAO.Create(room)
		created = append(created, roomNumber)
	}
	return created, nil
}

func (s *RoomService) GetRoom(id int64) (*model.Room, error) {
	return s.roomDAO.FindByID(id)
}

func (s *RoomService) UpdateRoom(id int64, roomTypeID *int64, status *string, floor *int) (*model.Room, error) {
	room, err := s.roomDAO.FindByID(id)
	if err != nil {
		return nil, err
	}
	if roomTypeID != nil {
		room.RoomTypeID = *roomTypeID
	}
	if status != nil {
		if *status == "maintenance" && room.Status == "occupied" {
			return nil, ErrRoomOccupied
		}
		room.Status = *status
	}
	if floor != nil {
		room.Floor = *floor
	}
	err = s.roomDAO.Update(room)
	return room, err
}

func (s *RoomService) DeleteRoom(id int64) error {
	room, err := s.roomDAO.FindByID(id)
	if err != nil {
		return err
	}
	if room.Status != "available" && room.Status != "maintenance" {
		return ErrRoomInUse
	}
	return s.roomDAO.Delete(id)
}

// ── 房态日历 ──

type RoomCalendarDay struct {
	ID         int64             `json:"id"`
	RoomNumber string            `json:"room_number"`
	RoomType   string            `json:"room_type_name"`
	Floor      int               `json:"floor"`
	Statuses   map[string]string `json:"statuses"`
}

type RoomCalendar struct {
	HotelID      int64             `json:"hotel_id"`
	Month        string            `json:"month"`
	Days         []int             `json:"days"`
	FirstWeekday int               `json:"first_weekday"`
	Rooms        []RoomCalendarDay `json:"rooms"`
}

func (s *RoomService) GetCalendar(hotelID int64, year, month int) (*RoomCalendar, error) {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()

	rooms, err := s.roomDAO.FindAllByHotel(hotelID)
	if err != nil {
		return nil, err
	}

	orders, err := s.orderDAO.FindByMonth(hotelID, firstDay, lastDay)
	if err != nil {
		return nil, err
	}

	cal := &RoomCalendar{
		HotelID:      hotelID,
		Month:        fmt.Sprintf("%d-%02d", year, month),
		Days:         make([]int, daysInMonth),
		FirstWeekday: int(firstDay.Weekday()), // Go: 0=Sunday -> adjusted in handler
		Rooms:        make([]RoomCalendarDay, 0, len(rooms)),
	}
	for d := 0; d < daysInMonth; d++ {
		cal.Days[d] = d + 1
	}

	for _, room := range rooms {
		day := RoomCalendarDay{
			ID:         room.ID,
			RoomNumber: room.RoomNumber,
			Floor:      room.Floor,
			Statuses:   make(map[string]string),
		}
		if room.RoomType != nil {
			day.RoomType = room.RoomType.Name
		}
		for d := 1; d <= daysInMonth; d++ {
			dayDate := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.UTC)
			key := fmt.Sprintf("%d", d)
			if room.Status == "maintenance" {
				day.Statuses[key] = "maintenance"
			} else {
				booked := false
				for _, o := range orders {
					if o.RoomID == room.ID && !o.CheckIn.After(dayDate) && o.CheckOut.After(dayDate) {
						if o.Status == "checked_in" {
							day.Statuses[key] = "occupied"
						} else {
							day.Statuses[key] = "booked"
						}
						booked = true
						break
					}
				}
				if !booked {
					day.Statuses[key] = "available"
				}
			}
		}
		cal.Rooms = append(cal.Rooms, day)
	}
	return cal, nil
}

// ── Excel 导出 ──

func (s *RoomService) ExportOrdersExcel(hotelID int64, startDate, endDate string, status string) ([]byte, string, error) {
	hotel, err := s.hotelDAO.FindByID(hotelID)
	if err != nil {
		return nil, "", err
	}
	orders, err := s.orderDAO.FindByHotelAndDateRange(hotelID, startDate, endDate, status)
	if err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	sheet := "订单报表"

	// Header
	f.SetCellValue(sheet, "A1", "订单号")
	f.SetCellValue(sheet, "B1", "入住人")
	f.SetCellValue(sheet, "C1", "房间号")
	f.SetCellValue(sheet, "D1", "房型")
	f.SetCellValue(sheet, "E1", "入住日期")
	f.SetCellValue(sheet, "F1", "退房日期")
	f.SetCellValue(sheet, "G1", "总价")
	f.SetCellValue(sheet, "H1", "状态")
	f.SetCellValue(sheet, "I1", "下单时间")

	statusCN := map[string]string{"pending": "待支付", "paid": "已支付", "checked_in": "已入住", "checked_out": "已退房", "cancelled": "已取消"}

	for i, o := range orders {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), o.OrderNo)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), o.GuestName)
		if o.Room != nil {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), o.Room.RoomNumber)
			if o.Room.RoomType != nil {
				f.SetCellValue(sheet, fmt.Sprintf("D%d", row), o.Room.RoomType.Name)
			}
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), o.CheckIn.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), o.CheckOut.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), o.TotalPrice)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), statusCN[o.Status])
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), o.CreatedAt.Format("2006-01-02 15:04"))
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("%s_报表_%s.xlsx", hotel.Name, time.Now().Format("2006-01-02"))
	return buf.Bytes(), filename, nil
}
