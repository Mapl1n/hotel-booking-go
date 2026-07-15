package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService *service.AdminService
	roomService  *service.RoomService
	userDAO      *dao.UserDAO
}

func NewAdminHandler(adminService *service.AdminService, roomService *service.RoomService, userDAO *dao.UserDAO) *AdminHandler {
	return &AdminHandler{adminService: adminService, roomService: roomService, userDAO: userDAO}
}

func (h *AdminHandler) requireHotelScope(c *gin.Context, hotelID int64) bool {
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

// Dashboard 数据看板
func (h *AdminHandler) Dashboard(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	stats, err := h.adminService.GetDashboard(hotelID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "酒店不存在")
		return
	}
	response.Success(c, stats)
}

// ListStaff 酒店员工列表
func (h *AdminHandler) ListStaff(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	users, err := h.userDAO.FindByHotelID(hotelID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	type userInfo struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		HotelID   *int64 `json:"hotel_id"`
		HotelName string `json:"hotel_name"`
		IsActive  bool   `json:"is_active"`
	}
	var list []userInfo
	for _, u := range users {
		hn := ""
		if u.Hotel != nil {
			hn = u.Hotel.Name
		}
		list = append(list, userInfo{u.ID, u.Username, u.Role, u.HotelID, hn, u.IsActive})
	}
	response.Success(c, list)
}

// ToggleStaff 启用/禁用员工
func (h *AdminHandler) ToggleStaff(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	currentUser := middleware.GetCurrentUser(c)
	if currentUser.Role == "hotel_admin" {
		target, err := h.userDAO.FindByID(userID)
		if err != nil || target.HotelID == nil || *target.HotelID != *currentUser.HotelID {
			response.Error(c, http.StatusForbidden, "只能操作自家酒店员工")
			return
		}
		if target.ID == currentUser.ID {
			response.Error(c, http.StatusBadRequest, "不能操作自己")
			return
		}
	}
	user, err := h.adminService.ToggleStaffActive(userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "用户不存在")
		return
	}
	msg := "员工已启用"
	if !user.IsActive {
		msg = "员工已禁用"
	}
	response.Success(c, gin.H{"message": msg, "is_active": user.IsActive})
}

// DeleteStaff 删除员工
func (h *AdminHandler) DeleteStaff(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	currentUser := middleware.GetCurrentUser(c)
	if userID == currentUser.ID {
		response.Error(c, http.StatusBadRequest, "不能删除自己")
		return
	}
	// hotel_admin can only delete own hotel staff
	if currentUser.Role == "hotel_admin" {
		target, err := h.userDAO.FindByID(userID)
		if err != nil || target.HotelID == nil || *target.HotelID != *currentUser.HotelID {
			response.Error(c, http.StatusForbidden, "只能删除自家酒店员工")
			return
		}
	}
	if err := h.adminService.DeleteStaff(userID); err != nil {
		response.Error(c, http.StatusNotFound, "用户不存在")
		return
	}
	response.SuccessMsg(c, "员工已删除")
}

// ExportOrders 导出 Excel 报表
func (h *AdminHandler) ExportOrders(c *gin.Context) {
	hotelID, _ := strconv.ParseInt(c.Query("hotel_id"), 10, 64)
	if !h.requireHotelScope(c, hotelID) { return }
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	status := c.Query("status")

	data, filename, err := h.roomService.ExportOrdersExcel(hotelID, startDate, endDate, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// HotelUpdateHandler 编辑酒店（超管）
func (h *AdminHandler) HotelUpdateHandler(c *gin.Context) {
	response.SuccessMsg(c, "请使用 /api/hotels/:id 接口")
}
