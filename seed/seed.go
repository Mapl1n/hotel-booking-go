package seed

import (
	"log"
	"time"

	"hotel-booking-go/internal/config"
	"hotel-booking-go/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Run(db *gorm.DB) {
	// ── 平台超管 ──
	var admin model.User
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username:     "admin",
			PasswordHash: string(hashed),
			Phone:        "10000000000",
			Role:         "super_admin",
			IsActive:     true,
		})
		log.Println("[SEED] 平台超管: admin / 123456")
	}

	// ── 示例酒店 ──
	var hotel model.Hotel
	if err := db.Where("name = ?", "云栖度假酒店").First(&hotel).Error; err != nil {
		hotel = model.Hotel{
			Name:        "云栖度假酒店",
			Address:     "杭州市西湖区龙井路88号",
			Phone:       "0571-88886666",
			Description: "坐落在西湖风景区内的精品度假酒店，拥有各类客房80间。",
		}
		db.Create(&hotel)

		// 酒店管理员
		h1, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username:     "13900001111",
			PasswordHash: string(h1),
			Phone:        "13900001111",
			Role:         "hotel_admin",
			HotelID:      &hotel.ID,
			IsActive:     true,
		})

		// 前台员工
		h2, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username:     "13900002222",
			PasswordHash: string(h2),
			Phone:        "13900002222",
			Role:         "front_desk",
			HotelID:      &hotel.ID,
			IsActive:     true,
		})

		// 房型
		rts := []model.RoomType{
			{HotelID: hotel.ID, Name: "标准大床房", Description: "1.8米大床，适合1-2人", Price: 288, Capacity: 2, IsActive: true},
			{HotelID: hotel.ID, Name: "豪华双床房", Description: "两张1.5米床，带阳台", Price: 488, Capacity: 4, IsActive: true},
			{HotelID: hotel.ID, Name: "行政套房", Description: "一室一厅，商务首选", Price: 888, Capacity: 2, IsActive: true},
			{HotelID: hotel.ID, Name: "家庭亲子房", Description: "卡通主题，含儿童床", Price: 588, Capacity: 3, IsActive: true},
		}
		db.Create(&rts)

		// 房间
		roomsData := []struct {
			rtID       int64
			roomNumber string
			floor      int
		}{
			{rts[0].ID, "101", 1}, {rts[0].ID, "102", 1}, {rts[0].ID, "103", 1},
			{rts[0].ID, "104", 1}, {rts[0].ID, "105", 1},
			{rts[0].ID, "201", 2}, {rts[0].ID, "202", 2}, {rts[0].ID, "203", 2},
			{rts[1].ID, "205", 2}, {rts[1].ID, "206", 2}, {rts[1].ID, "207", 2}, {rts[1].ID, "208", 2},
			{rts[2].ID, "301", 3}, {rts[2].ID, "302", 3},
			{rts[3].ID, "303", 3}, {rts[3].ID, "304", 3},
		}
		for _, rd := range roomsData {
			db.Create(&model.Room{
				HotelID: hotel.ID, RoomTypeID: rd.rtID,
				RoomNumber: rd.roomNumber, Floor: rd.floor, Status: "available",
			})
		}
		log.Printf("[SEED] 示例酒店: %s (16间房, 4种房型)", hotel.Name)
		log.Println("[SEED] 酒店管理员: 13900001111 / 123456")
		log.Println("[SEED] 前台员工:   13900002222 / 123456")
	}

	// ── 演示住客 ──
	var guest model.User
	if err := db.Where("username = ?", "13800008888").First(&guest).Error; err != nil {
		h, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username:     "13800008888",
			PasswordHash: string(h),
			Phone:        "13800008888",
			Role:         "guest",
			IsActive:     true,
		})
		log.Println("[SEED] 演示住客:   13800008888 / 123456")
	}
}

// AutoCancelExpiredOrders cron job: cancel unpaid orders after 30min
func AutoCancelExpiredOrders(db *gorm.DB, _ *config.Config) {
	cutoff := time.Now().Add(-30 * time.Minute)
	var orders []model.Order
	result := db.Preload("Room").Where(
		"status = ? AND created_at < ?", "pending", cutoff,
	).Find(&orders)
	if result.Error != nil {
		return
	}
	for _, o := range orders {
		db.Transaction(func(tx *gorm.DB) error {
			now := time.Now()
			o.Status = "cancelled"
			o.CancelledAt = &now
			o.CancelReason = "超时未支付（30分钟）系统自动取消"
			return tx.Save(&o).Error
		})
	}
	if len(orders) > 0 {
		log.Printf("[AUTO-CANCEL] 取消 %d 个超时订单", len(orders))
	}
}
