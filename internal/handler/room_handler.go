package handler

import (
	"net/http"
	"strconv"
	"time"

	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

// requireHotelScope hotel_admin 只能操作自家酒店
func (h *RoomHandler) requireHotelScope(c *gin.Context, hotelID int64) bool {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return false
	}
	if user.Role == "super_admin" {
		return true
	}
	if user.HotelID != nil && *user.HotelID == hotelID {
		return true
	}
	response.Error(c, http.StatusForbidden, "只能操作自家酒店")
	return false
}

// ── 房型 ──

type createRoomTypeReq struct {
	Name        string  `json:"name" binding:"required"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Capacity    int     `json:"capacity"`
	Description string  `json:"description"`
}

type updateRoomTypeReq struct {
	Name        *string  `json:"name"`
	Price       *float64 `json:"price"`
	Capacity    *int     `json:"capacity"`
	Description *string  `json:"description"`
	IsActive    *bool    `json:"is_active"`
}

// ListRoomTypes 房型列表
func (h *RoomHandler) ListRoomTypes(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	types, err := h.roomService.ListRoomTypes(hotelID, true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, types)
}

// CreateRoomType 添加房型
func (h *RoomHandler) CreateRoomType(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	var req createRoomTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	rt, err := h.roomService.CreateRoomType(hotelID, req.Name, req.Price, req.Capacity, req.Description)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(c, rt)
}

// UpdateRoomType 修改房型
func (h *RoomHandler) UpdateRoomType(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("type_id"), 10, 64)
	var req updateRoomTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	rt, err := h.roomService.UpdateRoomType(id, req.Name, req.Price, req.Capacity, req.Description, req.IsActive)
	if err != nil {
		response.Error(c, http.StatusNotFound, "房型不存在")
		return
	}
	response.Success(c, rt)
}

// ── 房间 ──

type createRoomReq struct {
	RoomTypeID int64  `json:"room_type_id" binding:"required"`
	RoomNumber string `json:"room_number" binding:"required"`
	Floor      int    `json:"floor"`
}

type batchCreateRoomReq struct {
	RoomTypeID  int64  `json:"room_type_id" binding:"required"`
	Floor       int    `json:"floor"`
	StartNumber int    `json:"start_number" binding:"required"`
	Count       int    `json:"count" binding:"required,gt=0,lte=50"`
	Prefix      string `json:"prefix"`
}

type updateRoomReq struct {
	RoomTypeID *int64  `json:"room_type_id"`
	Status     *string `json:"status" binding:"omitempty,oneof=available maintenance"`
	Floor      *int    `json:"floor"`
}

// GetAvailableRooms 按日期查询可用房间
func (h *RoomHandler) GetAvailableRooms(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	checkIn, _ := time.Parse("2006-01-02", c.Query("check_in"))
	checkOut, _ := time.Parse("2006-01-02", c.Query("check_out"))

	var roomTypeID *int64
	if v := c.Query("room_type_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		roomTypeID = &id
	}

	rooms, err := h.roomService.GetAvailableRooms(hotelID, checkIn, checkOut, roomTypeID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rooms)
}

// GetAllRooms 酒店全部房间
func (h *RoomHandler) GetAllRooms(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	rooms, err := h.roomService.GetAllRooms(hotelID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rooms)
}

// CreateRoom 添加房间
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	var req createRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	room, err := h.roomService.CreateRoom(hotelID, req.RoomTypeID, req.RoomNumber, req.Floor)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(c, room)
}

// BatchCreateRooms 批量添加房间
func (h *RoomHandler) BatchCreateRooms(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	var req batchCreateRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	created, err := h.roomService.BatchCreateRooms(hotelID, req.RoomTypeID, req.Floor, req.StartNumber, req.Count, req.Prefix)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "成功创建 " + strconv.Itoa(len(created)) + " 间房间", "rooms": created})
}

// UpdateRoom 修改房间
func (h *RoomHandler) UpdateRoom(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("room_id"), 10, 64)
	// hotel scope check
	r, err := h.roomService.GetRoom(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "房间不存在")
		return
	}
	if !h.requireHotelScope(c, r.HotelID) { return }

	var req updateRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	room, err := h.roomService.UpdateRoom(id, req.RoomTypeID, req.Status, req.Floor)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, room)
}

// DeleteRoom 删除房间
func (h *RoomHandler) DeleteRoom(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("room_id"), 10, 64)
	// hotel scope check
	r, err := h.roomService.GetRoom(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "房间不存在")
		return
	}
	if !h.requireHotelScope(c, r.HotelID) { return }

	if err := h.roomService.DeleteRoom(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessMsg(c, "房间已删除")
}

// GetCalendar 房态日历
func (h *RoomHandler) GetCalendar(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	month := c.Query("month") // YYYY-MM
	y, m := time.Now().Year(), int(time.Now().Month())
	if month != "" {
		t, err := time.Parse("2006-01", month)
		if err == nil {
			y, m = t.Year(), int(t.Month())
		}
	}
	cal, err := h.roomService.GetCalendar(hotelID, y, m)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cal)
}
