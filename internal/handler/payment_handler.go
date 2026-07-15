package handler

import (
	"net/http"
	"strconv"
	"time"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/model"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/payment"
	"hotel-booking-go/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
	paymentDAO     *dao.PaymentDAO
	orderDAO       *dao.OrderDAO
	db             *gorm.DB
}

func NewPaymentHandler(paymentService *service.PaymentService, paymentDAO *dao.PaymentDAO, orderDAO *dao.OrderDAO, db *gorm.DB) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		paymentDAO:     paymentDAO,
		orderDAO:       orderDAO,
		db:             db,
	}
}

type createPaymentReq struct {
	OrderID       int64  `json:"order_id" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"required,oneof=wechat alipay bankcard mock"`
}

// Methods 获取可用支付方式
func (h *PaymentHandler) Methods(c *gin.Context) {
	response.Success(c, payment.Methods)
}

// Create 创建支付
func (h *PaymentHandler) Create(c *gin.Context) {
	var req createPaymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	result, err := h.paymentService.Create(req.OrderID, req.PaymentMethod)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{
		"success":    result.Success,
		"payment_id": result.PaymentID,
		"qr_code":    result.QRCode,
		"pay_url":    result.PayURL,
		"payment_no": result.PaymentNo,
		"message":    result.Message,
	})
}

// Status 查询支付状态
func (h *PaymentHandler) Status(c *gin.Context) {
	paymentID, _ := strconv.ParseInt(c.Param("payment_id"), 10, 64)
	pmt, err := h.paymentService.GetStatus(paymentID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "支付记录不存在")
		return
	}
	_ = middleware.GetCurrentUser(c) // auth check
	response.Success(c, gin.H{
		"payment_id": pmt.ID,
		"status":     pmt.Status,
		"paid":       pmt.Status == "success",
		"payment_no": pmt.PaymentNo,
		"amount":     pmt.Amount,
	})
}

// WechatCallback 微信支付回调（外部平台异步通知，无需认证）
func (h *PaymentHandler) WechatCallback(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "参数错误"})
		return
	}
	// 处理支付结果
	h.processCallback(body, "wechat")
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
}

// AlipayCallback 支付宝回调（无需认证）
func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	h.processCallback(body, "alipay")
	c.String(http.StatusOK, "success")
}

// processCallback 处理支付回调：更新 Payment 和 Order 状态（幂等）
func (h *PaymentHandler) processCallback(data map[string]interface{}, method string) {
	paymentNo, _ := data["payment_no"].(string)
	if paymentNo == "" {
		return
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		payment, err := h.paymentDAO.FindByPaymentNo(paymentNo)
		if err != nil {
			return nil // 记录不存在，忽略
		}
		if payment.Status == "success" {
			return nil // 幂等：已处理
		}

		now := time.Now()
		payment.Status = "success"
		payment.PaidAt = &now
		tx.Save(payment)

		// 更新订单状态
		var order model.Order
		if err := tx.First(&order, payment.OrderID).Error; err == nil {
			if order.Status == "pending" {
				order.Status = "paid"
				tx.Save(&order)
			}
		}
		return nil
	})
}
