package handler

import (
	"net/http"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
	userDAO     *dao.UserDAO
}

func NewAuthHandler(authService *service.AuthService, userDAO *dao.UserDAO) *AuthHandler {
	return &AuthHandler{authService: authService, userDAO: userDAO}
}

type sendCodeReq struct {
	Phone string `json:"phone" binding:"required,len=11"`
}

type registerReq struct {
	Username string `json:"username" binding:"required,len=11"`
	Password string `json:"password" binding:"required,min=8,max=20"`
	Code     string `json:"code" binding:"required,len=6"`
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type createStaffReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=hotel_admin front_desk"`
	HotelID  int64  `json:"hotel_id" binding:"required"`
	Phone    string `json:"phone"`
}

// SendCode 发送短信验证码
func (h *AuthHandler) SendCode(c *gin.Context) {
	var req sendCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "手机号格式错误")
		return
	}
	if err := h.authService.SendCode(c.Request.Context(), req.Phone); err != nil {
		response.Error(c, http.StatusTooManyRequests, err.Error())
		return
	}
	response.SuccessMsg(c, "验证码已发送")
}

// Register 住客注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.authService.Register(c.Request.Context(), req.Username, req.Password, req.Code); err != nil {
		switch err {
		case service.ErrPhoneExists:
			response.Error(c, http.StatusBadRequest, err.Error())
		case service.ErrInvalidCode:
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	response.SuccessMsg(c, "注册成功")
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	ip := c.ClientIP()
	token, user, err := h.authService.Login(req.Username, req.Password, ip)
	if err != nil {
		switch err {
		case service.ErrInvalidLogin:
			response.Error(c, http.StatusUnauthorized, err.Error())
		case service.ErrAccountDisabled:
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "登录失败")
		}
		return
	}

	hotelName := ""
	if user.Hotel != nil {
		hotelName = user.Hotel.Name
	}
	response.Success(c, gin.H{
		"access_token": token,
		"token_type":   "bearer",
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"hotel_id":   user.HotelID,
			"hotel_name": hotelName,
			"is_active":  user.IsActive,
		},
	})
}

// GetMe 获取当前用户信息（从 DB 加载完整信息含酒店名称）
func (h *AuthHandler) GetMe(c *gin.Context) {
	jwtUser := middleware.GetCurrentUser(c)
	if jwtUser == nil {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}
	// 从 DB 获取完整用户（含 Hotel 关联）
	user, err := h.userDAO.FindByID(jwtUser.ID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "用户不存在")
		return
	}
	hotelName := ""
	if user.Hotel != nil {
		hotelName = user.Hotel.Name
	}
	response.Success(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"hotel_id":   user.HotelID,
		"hotel_name": hotelName,
		"is_active":  user.IsActive,
	})
}

// CreateStaff 创建员工账号
func (h *AuthHandler) CreateStaff(c *gin.Context) {
	var req createStaffReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	currentUser := middleware.GetCurrentUser(c)
	if currentUser.Role == "hotel_admin" {
		if req.Role != "front_desk" {
			response.Error(c, http.StatusForbidden, "酒店管理员只能创建前台员工账号")
			return
		}
		req.HotelID = *currentUser.HotelID
	}
	user, err := h.authService.CreateStaff(req.Username, req.Password, req.Role, req.HotelID, req.Phone)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "员工账号创建成功", "user_id": user.ID, "role": user.Role})
}

// GetLoginLogs 查询登录日志
func (h *AuthHandler) GetLoginLogs(c *gin.Context) {
	currentUser := middleware.GetCurrentUser(c)
	var userIDs []int64
	if currentUser.Role == "hotel_admin" {
		// TODO: get all user IDs for the hotel
	}
	logs, err := h.authService.GetLoginLogs(userIDs, 50)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, logs)
}
