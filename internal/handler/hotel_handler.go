package handler

import (
	"net/http"
	"strconv"

	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type HotelHandler struct {
	hotelService *service.HotelService
}

func NewHotelHandler(hotelService *service.HotelService) *HotelHandler {
	return &HotelHandler{hotelService: hotelService}
}

// checkHotelScope hotel_admin 只能操作自家酒店
func checkHotelScope(c *gin.Context, hotelID int64) bool {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return false
	}
	if user.Role == "super_admin" {
		return true
	}
	if user.Role == "hotel_admin" && user.HotelID != nil && *user.HotelID == hotelID {
		return true
	}
	response.Error(c, http.StatusForbidden, "只能操作自家酒店")
	return false
}

type createHotelReq struct {
	Name        string `json:"name" binding:"required"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

type updateHotelReq struct {
	Name        *string `json:"name"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	Description *string `json:"description"`
}

// List 酒店列表（公开）
func (h *HotelHandler) List(c *gin.Context) {
	hotels, err := h.hotelService.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, hotels)
}

// Get 酒店详情
func (h *HotelHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("hotel_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	hotel, err := h.hotelService.Get(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "酒店不存在")
		return
	}
	response.Success(c, hotel)
}

// Create 创建酒店（超管）
func (h *HotelHandler) Create(c *gin.Context) {
	var req createHotelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	hotel, err := h.hotelService.Create(req.Name, req.Address, req.Phone, req.Description)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Created(c, hotel)
}

// Update 编辑酒店信息（超管或本店管理员）
func (h *HotelHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("hotel_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if !checkHotelScope(c, id) {
		return
	}
	var req updateHotelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	hotel, err := h.hotelService.Update(id, req.Name, req.Address, req.Phone, req.Description)
	if err != nil {
		response.Error(c, http.StatusNotFound, "酒店不存在")
		return
	}
	response.Success(c, hotel)
}
