package service

import (
	"errors"
	"fmt"
	"time"

	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/model"
	"hotel-booking-go/pkg/crypto"

	"gorm.io/gorm"
)

type OrderService struct {
	orderDAO   *dao.OrderDAO
	roomDAO    *dao.RoomDAO
	paymentDAO *dao.PaymentDAO
	hotelDAO   *dao.HotelDAO
	db         *gorm.DB
}

func NewOrderService(orderDAO *dao.OrderDAO, roomDAO *dao.RoomDAO, paymentDAO *dao.PaymentDAO, hotelDAO *dao.HotelDAO, db *gorm.DB) *OrderService {
	return &OrderService{
		orderDAO:   orderDAO,
		roomDAO:    roomDAO,
		paymentDAO: paymentDAO,
		hotelDAO:   hotelDAO,
		db:         db,
	}
}

var (
	ErrRoomNotAvailable    = errors.New("房间不可用或不存在")
	ErrDateConflict        = errors.New("该时间段房间已被预订")
	ErrInvalidDateRange    = errors.New("入住日期必须早于退房日期")
	ErrPastDate            = errors.New("入住日期不能是过去")
	ErrOrderNotFound       = errors.New("订单不存在")
	ErrOrderNotCancellable = errors.New("当前状态不可取消")
	ErrOrderMustPayFirst   = errors.New("请先完成支付后再办理入住")
	ErrOrderNotCheckin     = errors.New("当前状态不可退房")
	ErrOrderNotExtensible  = errors.New("仅已入住的订单可以办理续住")
	ErrExtendConflict      = errors.New("续住时段与后续预订冲突")
	ErrUnauthorized        = errors.New("无权访问此订单")
)

const AutoCancelMinutes = 30

// CreateOrder ★ 核心：事务 + SELECT FOR UPDATE 行锁并发控制
//
// 并发保护机制:
//  1. 开启数据库事务 (READ COMMITTED 或更高隔离级别)
//  2. SELECT ... FOR UPDATE 悲观行锁锁定目标房间行
//  3. 在事务内检查时间段冲突 (行锁保证其他事务等待)
//  4. 订单号使用事务内 MAX + 1 (避免 LIKE COUNT 的竞态)
//  5. 创建订单在同一事务中，原子性保证
func (s *OrderService) CreateOrder(userID int64, roomID int64, guestName, idCard string, checkIn, checkOut time.Time) (*model.Order, error) {
	// 基础校验
	if !checkIn.Before(checkOut) {
		return nil, ErrInvalidDateRange
	}
	today := time.Now().Truncate(24 * time.Hour)
	if checkIn.Before(today) {
		return nil, ErrPastDate
	}

	// 加密身份证
	encryptedIDCard, err := crypto.Encrypt(idCard)
	if err != nil {
		return nil, fmt.Errorf("身份证加密失败: %w", err)
	}

	var order *model.Order

	// ★ 数据库事务：行锁 + 冲突检查 + 订单创建 原子执行
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 1. SELECT ... FOR UPDATE 悲观行锁 → 阻塞其他并发事务
		room, err := s.roomDAO.FindByIDForUpdate(tx, roomID)
		if err != nil {
			return ErrRoomNotAvailable
		}

		// 2. 在行锁保护下检查时间段冲突
		conflict, err := s.orderDAO.CountDateConflict(tx, roomID, checkIn, checkOut)
		if err != nil {
			return err
		}
		if conflict > 0 {
			return ErrDateConflict
		}

		// 3. 生成订单号：使用 MAX 而非 COUNT（避免并发读到相同计数）
		//    注意：在极端并发下仍可能碰撞，依赖 UNIQUE 约束兜底
		orderNo, err := s.genOrderNo(tx)
		if err != nil {
			return err
		}

		// 4. 计算价格
		nights := int(checkOut.Sub(checkIn).Hours() / 24)
		totalPrice := room.RoomType.Price * float64(nights)

		// 5. 创建订单
		order = &model.Order{
			HotelID:    room.HotelID,
			UserID:     userID,
			RoomID:     roomID,
			OrderNo:    orderNo,
			GuestName:  guestName,
			IDCard:     encryptedIDCard,
			CheckIn:    checkIn,
			CheckOut:   checkOut,
			TotalPrice: totalPrice,
			Status:     "pending",
		}
		return tx.Create(order).Error
	})

	if err != nil {
		return nil, err
	}
	return order, nil
}

// genOrderNo 生成订单号 ORD + YYYYMMDD + 4位流水
// 使用 MAX 而非 COUNT 减少竞态，但依赖 UNIQUE 约束兜底
func (s *OrderService) genOrderNo(tx *gorm.DB) (string, error) {
	today := time.Now().Format("20060102")
	prefix := "ORD" + today

	var lastNo string
	err := tx.Model(&model.Order{}).
		Where("order_no LIKE ?", prefix+"%").
		Order("order_no DESC").
		Limit(1).
		Pluck("order_no", &lastNo).Error
	if err != nil {
		return "", err
	}

	seq := 1
	if lastNo != "" && len(lastNo) >= len(prefix)+4 {
		fmt.Sscanf(lastNo[len(prefix):], "%04d", &seq)
		seq++
	}
	if seq > 9999 {
		seq = 9999 // 超过后重置，UNIQUE 约束会拒绝重复
	}

	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

// CheckAuth 订单权限校验
func (s *OrderService) CheckAuth(order *model.Order, user *model.User) error {
	if user.Role == "super_admin" {
		return nil
	}
	if user.Role == "hotel_admin" || user.Role == "front_desk" {
		if user.HotelID == nil || order.HotelID != *user.HotelID {
			return ErrUnauthorized
		}
		return nil
	}
	if order.UserID != user.ID {
		return ErrUnauthorized
	}
	return nil
}

// CancelOrder 取消订单（事务内更新房间状态）
func (s *OrderService) CancelOrder(orderID int64, reason string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.orderDAO.FindByID(orderID)
		if err != nil {
			return ErrOrderNotFound
		}
		if order.Status != "pending" && order.Status != "paid" {
			return ErrOrderNotCancellable
		}
		// 已支付订单取消时释放房间（在事务内用 tx 更新）
		if order.Status == "paid" {
			if err := s.roomDAO.UpdateWithTx(tx, &model.Room{
				ID:     order.RoomID,
				Status: "available",
			}); err != nil {
				return err
			}
		}
		now := time.Now()
		order.Status = "cancelled"
		order.CancelledAt = &now
		order.CancelReason = reason
		return s.orderDAO.Update(tx, order)
	})
}

// CheckIn 办理入住（事务内更新房间状态）
func (s *OrderService) CheckIn(orderID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.orderDAO.FindByID(orderID)
		if err != nil {
			return ErrOrderNotFound
		}
		if order.Status != "paid" {
			return ErrOrderMustPayFirst
		}
		order.Status = "checked_in"

		// ★ 在事务内用 tx 更新房间状态
		if err := s.roomDAO.UpdateWithTx(tx, &model.Room{
			ID:     order.RoomID,
			Status: "occupied",
		}); err != nil {
			return err
		}
		return s.orderDAO.Update(tx, order)
	})
}

// CheckOut 办理退房（事务内释放房间）
func (s *OrderService) CheckOut(orderID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.orderDAO.FindByID(orderID)
		if err != nil {
			return ErrOrderNotFound
		}
		if order.Status != "checked_in" {
			return ErrOrderNotCheckin
		}
		order.Status = "checked_out"

		// ★ 在事务内用 tx 释放房间
		if err := s.roomDAO.UpdateWithTx(tx, &model.Room{
			ID:     order.RoomID,
			Status: "available",
		}); err != nil {
			return err
		}
		return s.orderDAO.Update(tx, order)
	})
}

// ExtendStay 续住（事务内校验 + 安全的价格计算）
func (s *OrderService) ExtendStay(orderID int64, extraDays int) (*OrderExtendResult, error) {
	var result *OrderExtendResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.orderDAO.FindByID(orderID)
		if err != nil {
			return ErrOrderNotFound
		}
		if order.Status != "checked_in" {
			return ErrOrderNotExtensible
		}
		if extraDays <= 0 {
			return errors.New("续住天数必须大于0")
		}

		// ★ 安全：检查 Room 和 RoomType 是否为 nil
		if order.Room == nil || order.Room.RoomType == nil {
			return errors.New("房间信息缺失，请联系管理员")
		}

		newCheckOut := order.CheckOut.AddDate(0, 0, extraDays)

		// 检查续住时段是否冲突
		conflict, err := s.orderDAO.CountDateConflict(tx, order.RoomID, order.CheckOut, newCheckOut, order.ID)
		if err != nil {
			return err
		}
		if conflict > 0 {
			return ErrExtendConflict
		}

		extraPrice := order.Room.RoomType.Price * float64(extraDays)
		order.CheckOut = newCheckOut
		order.TotalPrice += extraPrice

		if err := s.orderDAO.Update(tx, order); err != nil {
			return err
		}

		result = &OrderExtendResult{
			ExtraDays:   extraDays,
			ExtraPrice:  extraPrice,
			NewTotal:    order.TotalPrice,
			NewCheckOut: newCheckOut,
		}
		return nil
	})
	return result, err
}

type OrderExtendResult struct {
	ExtraDays   int       `json:"extra_days"`
	ExtraPrice  float64   `json:"extra_price"`
	NewTotal    float64   `json:"new_total_price"`
	NewCheckOut time.Time `json:"new_check_out"`
}

// ListOrders 用户/酒店员工查看订单列表
func (s *OrderService) ListOrders(user *model.User, hotelID *int64, status string) ([]model.Order, error) {
	if user.Role == "super_admin" || user.Role == "hotel_admin" || user.Role == "front_desk" {
		filterHotelID := user.HotelID
		if user.Role == "super_admin" && hotelID != nil {
			filterHotelID = hotelID
		}
		if filterHotelID != nil {
			return s.orderDAO.ListByHotel(*filterHotelID, status)
		}
		// super_admin without hotel filter: return all orders
		return s.orderDAO.ListAll(status)
	}
	return s.orderDAO.ListByUser(user.ID)
}

// GetOrderDetail 获取订单详情（含身份证解密+脱敏）
func (s *OrderService) GetOrderDetail(orderID int64, showFullID bool, isStaff bool) (map[string]interface{}, error) {
	order, err := s.orderDAO.FindByID(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	// 解密身份证
	plainID, err := crypto.Decrypt(order.IDCard)
	if err != nil {
		plainID = order.IDCard
	}
	masked := crypto.MaskIDCard(plainID)

	result := map[string]interface{}{
		"id":             order.ID,
		"order_no":       order.OrderNo,
		"user_id":        order.UserID,
		"guest_name":     order.GuestName,
		"id_card":        "",
		"id_card_masked": masked,
		"check_in":       order.CheckIn.Format("2006-01-02"),
		"check_out":      order.CheckOut.Format("2006-01-02"),
		"total_price":    order.TotalPrice,
		"status":         order.Status,
		"created_at":     order.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// 完整身份证仅酒店员工可查看
	if showFullID && isStaff {
		result["id_card"] = plainID
	}

	if order.CancelledAt != nil {
		result["cancelled_at"] = order.CancelledAt.Format("2006-01-02 15:04:05")
		result["cancel_reason"] = order.CancelReason
	}

	// 房间信息
	if order.Room != nil {
		roomData := map[string]interface{}{
			"id":          order.Room.ID,
			"room_number": order.Room.RoomNumber,
			"floor":       order.Room.Floor,
			"status":      order.Room.Status,
		}
		if order.Room.RoomType != nil {
			roomData["room_type"] = map[string]interface{}{
				"id":          order.Room.RoomType.ID,
				"name":        order.Room.RoomType.Name,
				"description": order.Room.RoomType.Description,
				"price":       order.Room.RoomType.Price,
				"capacity":    order.Room.RoomType.Capacity,
				"image_url":   order.Room.RoomType.ImageURL,
			}
		}
		result["room"] = roomData
	}

	if order.Hotel != nil {
		result["hotel_name"] = order.Hotel.Name
		result["hotel_address"] = order.Hotel.Address
		result["hotel_phone"] = order.Hotel.Phone
	}

	// 支付信息
	payment, err := s.paymentDAO.FindSuccessByOrder(orderID)
	if err == nil && payment != nil {
		payData := map[string]interface{}{
			"id":             payment.ID,
			"order_id":       payment.OrderID,
			"payment_no":     payment.PaymentNo,
			"amount":         payment.Amount,
			"payment_method": payment.PaymentMethod,
			"status":         payment.Status,
		}
		if payment.PaidAt != nil {
			payData["paid_at"] = payment.PaidAt.Format("2006-01-02 15:04:05")
		}
		result["payment"] = payData
	}

	return result, nil
}

// AutoCancelExpired 超时未支付自动取消（cron 调用）
func (s *OrderService) AutoCancelExpired() error {
	orders, err := s.orderDAO.FindExpiredPending(AutoCancelMinutes)
	if err != nil {
		return err
	}
	for _, o := range orders {
		s.db.Transaction(func(tx *gorm.DB) error {
			now := time.Now()
			o.Status = "cancelled"
			o.CancelledAt = &now
			o.CancelReason = fmt.Sprintf("超时未支付（%d分钟）系统自动取消", AutoCancelMinutes)
			return s.orderDAO.Update(tx, &o)
		})
	}
	return nil
}
