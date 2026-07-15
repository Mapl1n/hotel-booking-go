package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/model"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderService *service.OrderService
	orderDAO     *dao.OrderDAO
}

func NewOrderHandler(orderService *service.OrderService, orderDAO *dao.OrderDAO) *OrderHandler {
	return &OrderHandler{orderService: orderService, orderDAO: orderDAO}
}

// checkOrderAccess 校验当前用户是否有权操作此订单
func (h *OrderHandler) checkOrderAccess(c *gin.Context, orderID int64) (*model.Order, bool) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return nil, false
	}
	order, err := h.orderDAO.FindByID(orderID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "订单不存在")
		return nil, false
	}
	// 超管可以看所有
	if user.Role == "super_admin" {
		return order, true
	}
	// 酒店员工只能看自家酒店订单
	if user.Role == "hotel_admin" || user.Role == "front_desk" {
		if user.HotelID != nil && *user.HotelID == order.HotelID {
			return order, true
		}
		response.Error(c, http.StatusForbidden, "无权操作此订单")
		return nil, false
	}
	// 住客只能看自己的订单
	if order.UserID != user.ID {
		response.Error(c, http.StatusForbidden, "无权操作此订单")
		return nil, false
	}
	return order, true
}

type createOrderReq struct {
	RoomID    int64  `json:"room_id" binding:"required,gt=0"`
	GuestName string `json:"guest_name" binding:"required,min=2,max=20"`
	IDCard    string `json:"id_card" binding:"required,len=18"`
	CheckIn   string `json:"check_in" binding:"required"`  // "2006-01-02"
	CheckOut  string `json:"check_out" binding:"required"` // "2006-01-02"
}

type cancelOrderReq struct {
	Reason string `json:"reason"`
}

type payOrderReq struct {
	PaymentMethod string `json:"payment_method" binding:"required,oneof=wechat alipay bankcard"`
}

type extendStayReq struct {
	ExtraDays int `json:"extra_days" binding:"required,gt=0"`
}

// CreateOrder ★ 预订下单（核心接口）
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	checkIn, err := time.Parse("2006-01-02", req.CheckIn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "入住日期格式错误")
		return
	}
	checkOut, err := time.Parse("2006-01-02", req.CheckOut)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "退房日期格式错误")
		return
	}

	user := middleware.GetCurrentUser(c)
	order, err := h.orderService.CreateOrder(user.ID, req.RoomID, req.GuestName, req.IDCard, checkIn, checkOut)
	if err != nil {
		switch err {
		case service.ErrRoomNotAvailable:
			response.Error(c, http.StatusBadRequest, err.Error())
		case service.ErrDateConflict:
			response.Error(c, http.StatusBadRequest, err.Error())
		case service.ErrInvalidDateRange, service.ErrPastDate:
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "下单失败")
		}
		return
	}
	response.Success(c, gin.H{
		"message":     "预订成功",
		"order_id":    order.ID,
		"order_no":    order.OrderNo,
		"total_price": order.TotalPrice,
	})
}

// List 订单列表
func (h *OrderHandler) List(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var hotelID *int64
	if v := c.Query("hotel_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		hotelID = &id
	}
	status := c.Query("status")

	orders, err := h.orderService.ListOrders(user, hotelID, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	type OrderListItem struct {
		ID         int64   `json:"id"`
		OrderNo    string  `json:"order_no"`
		RoomNumber string  `json:"room_number"`
		HotelName  string  `json:"hotel_name"`
		GuestName  string  `json:"guest_name"`
		CheckIn    string  `json:"check_in"`
		CheckOut   string  `json:"check_out"`
		TotalPrice float64 `json:"total_price"`
		Status     string  `json:"status"`
		CreatedAt  string  `json:"created_at"`
	}

	var items []OrderListItem
	for _, o := range orders {
		item := OrderListItem{
			ID:         o.ID,
			OrderNo:    o.OrderNo,
			GuestName:  o.GuestName,
			CheckIn:    o.CheckIn.Format("2006-01-02"),
			CheckOut:   o.CheckOut.Format("2006-01-02"),
			TotalPrice: o.TotalPrice,
			Status:     o.Status,
			CreatedAt:  o.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if o.Room != nil {
			item.RoomNumber = o.Room.RoomNumber
		}
		if o.Hotel != nil {
			item.HotelName = o.Hotel.Name
		}
		items = append(items, item)
	}
	response.Success(c, items)
}

// GetDetail 订单详情
func (h *OrderHandler) GetDetail(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("order_id"), 10, 64)
	showFullID := c.Query("show_full_id") == "true"

	user := middleware.GetCurrentUser(c)
	if _, ok := h.checkOrderAccess(c, orderID); !ok { return }

	isStaff := user.Role == "super_admin" || user.Role == "hotel_admin" || user.Role == "front_desk"
	detail, err := h.orderService.GetOrderDetail(orderID, showFullID, isStaff)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, detail)
}

// Cancel 取消订单
func (h *OrderHandler) Cancel(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if _, ok := h.checkOrderAccess(c, orderID); !ok { return }
	var req cancelOrderReq
	c.ShouldBindJSON(&req)

	if err := h.orderService.CancelOrder(orderID, req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessMsg(c, "订单已取消")
}

// CheckIn 办理入住
func (h *OrderHandler) CheckIn(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if _, ok := h.checkOrderAccess(c, orderID); !ok { return }
	if err := h.orderService.CheckIn(orderID); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessMsg(c, "入住办理成功")
}

// CheckOut 办理退房
func (h *OrderHandler) CheckOut(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if _, ok := h.checkOrderAccess(c, orderID); !ok { return }
	if err := h.orderService.CheckOut(orderID); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessMsg(c, "退房办理成功")
}

// ExtendStay 续住
func (h *OrderHandler) ExtendStay(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if _, ok := h.checkOrderAccess(c, orderID); !ok { return }
	var req extendStayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	result, err := h.orderService.ExtendStay(orderID, req.ExtraDays)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{
		"message":          "续住办理成功",
		"extra_days":       result.ExtraDays,
		"extra_price":      result.ExtraPrice,
		"new_total_price":  result.NewTotal,
		"new_check_out":    result.NewCheckOut.Format("2006-01-02"),
	})
}

// PrintOrder 打印入住登记单
func (h *OrderHandler) PrintOrder(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if _, ok := h.checkOrderAccess(c, orderID); !ok { return }
	detail, err := h.orderService.GetOrderDetail(orderID, true, true)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	statusCN := map[string]string{"pending": "待支付", "paid": "已支付", "checked_in": "已入住", "checked_out": "已退房", "cancelled": "已取消"}

	hotelName, _ := detail["hotel_name"].(string)
	hotelAddr, _ := detail["hotel_address"].(string)
	hotelPhone, _ := detail["hotel_phone"].(string)
	orderNo, _ := detail["order_no"].(string)
	guestName, _ := detail["guest_name"].(string)
	status, _ := detail["status"].(string)
	checkIn, _ := detail["check_in"].(string)
	checkOut, _ := detail["check_out"].(string)
	totalPrice, _ := detail["total_price"].(float64)
	idCardMasked, _ := detail["id_card_masked"].(string)

	roomInfo := ""
	if r, ok := detail["room"].(map[string]interface{}); ok {
		rn, _ := r["room_number"].(string)
		fl, _ := r["floor"].(float64)
		rtName := ""
		if rt, ok := r["room_type"].(map[string]interface{}); ok {
			rtName, _ = rt["name"].(string)
		}
		roomInfo = fmt.Sprintf("%s（%s）%d楼", rn, rtName, int(fl))
	}

	pricePerNight := ""
	if r, ok := detail["room"].(map[string]interface{}); ok {
		if rt, ok := r["room_type"].(map[string]interface{}); ok {
			if p, ok := rt["price"].(float64); ok {
				pricePerNight = fmt.Sprintf("¥%.0f/晚", p)
			}
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><title>入住登记单 %s</title>
<style>
  @media print { body { -webkit-print-color-adjust: exact; } .no-print { display: none; } }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'SimSun', '宋体', serif; padding: 40px; color: #000; }
  .header { text-align: center; border-bottom: 2px solid #000; padding-bottom: 12px; margin-bottom: 20px; }
  .header h1 { font-size: 22px; }
  .row { display: flex; margin-bottom: 10px; font-size: 15px; }
  .label { width: 80px; font-weight: bold; }
  .value { flex: 1; border-bottom: 1px dashed #000; padding-bottom: 2px; }
  table { width: 100%%; border-collapse: collapse; margin: 12px 0; }
  td, th { border: 1px solid #000; padding: 6px 10px; font-size: 14px; }
  th { background: #e5e7eb; }
  .total { font-size: 18px; font-weight: bold; text-align: right; margin-top: 10px; }
  .footer { margin-top: 30px; font-size: 13px; }
  .sig { display: inline-block; width: 200px; border-bottom: 1px solid #000; }
  .btn { display: inline-block; padding: 10px 24px; background: #2563eb; color: #fff; border: none; border-radius: 6px; font-size: 14px; cursor: pointer; margin-top: 20px; text-decoration: none; }
</style></head>
<body>
<div class="no-print" style="text-align:center;margin-bottom:20px">
  <button class="btn" onclick="window.print()">🖨️ 打印入住单</button>
  <button class="btn" style="background:#6b7280;margin-left:8px" onclick="window.close()">关闭</button>
</div>
<div class="header"><h1>%s</h1><p>📍 %s  📞 %s</p><p style="font-size:16px;margin-top:8px;font-weight:bold">入住登记单</p></div>
<div><h3 style="font-size:16px;border-bottom:1px solid #000;padding-bottom:4px;margin-bottom:10px">订单信息</h3>
  <div class="row"><span class="label">订单号</span><span class="value">%s</span></div>
  <div class="row"><span class="label">订单状态</span><span class="value">%s</span></div>
</div>
<div><h3 style="font-size:16px;border-bottom:1px solid #000;padding-bottom:4px;margin:20px 0 10px">入住人信息</h3>
  <div class="row"><span class="label">姓　名</span><span class="value">%s</span></div>
  <div class="row"><span class="label">身份证号</span><span class="value">%s</span></div>
</div>
<div><h3 style="font-size:16px;border-bottom:1px solid #000;padding-bottom:4px;margin:20px 0 10px">房间信息</h3>
  <div class="row"><span class="label">房间号</span><span class="value">%s</span></div>
</div>
<div><h3 style="font-size:16px;border-bottom:1px solid #000;padding-bottom:4px;margin:20px 0 10px">入住信息</h3>
  <table><tr><th>入住日期</th><th>退房日期</th><th>单价</th><th>总价</th></tr>
  <tr><td>%s</td><td>%s</td><td>%s</td><td style="font-weight:bold">¥%.0f</td></tr></table>
  <div class="total">实付金额：¥%.0f</div>
</div>
<div class="footer">
  <p>客人签名：<span class="sig"></span></p>
  <p style="margin-top:12px">前台确认：<span class="sig"></span></p>
  <p style="margin-top:12px;font-size:12px;color:#888">打印时间：%s</p>
</div>
</body></html>`,
		orderNo,
		hotelName, hotelAddr, hotelPhone,
		orderNo, statusCN[status],
		guestName, idCardMasked,
		roomInfo,
		checkIn, checkOut, pricePerNight, totalPrice, totalPrice,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// Pay 订单支付（兼容接口，委托 PaymentService）
func (h *OrderHandler) Pay(c *gin.Context) {
	response.Error(c, http.StatusBadRequest, "请使用 /api/payment/create 接口")
}
