package router

import (
	"time"

	"hotel-booking-go/internal/config"
	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/handler"
	"hotel-booking-go/internal/middleware"
	"hotel-booking-go/internal/service"
	"hotel-booking-go/pkg/crypto"
	"hotel-booking-go/pkg/sms"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *gin.Engine {
	// Init crypto with secret key
	crypto.Init(cfg.JWTSecretKey)

	// ── DAOs ──
	userDAO := dao.NewUserDAO(db)
	hotelDAO := dao.NewHotelDAO(db)
	roomDAO := dao.NewRoomDAO(db)
	roomTypeDAO := dao.NewRoomTypeDAO(db)
	orderDAO := dao.NewOrderDAO(db)
	paymentDAO := dao.NewPaymentDAO(db)
	loginLogDAO := dao.NewLoginLogDAO(db)

	// ── SMS ──
	smsService := sms.NewService(rdb, cfg.SMSProvider, cfg.SMSMockCode)

	// ── Services ──
	authService := service.NewAuthService(userDAO, loginLogDAO, smsService, cfg)
	hotelService := service.NewHotelService(hotelDAO)
	roomService := service.NewRoomService(roomDAO, roomTypeDAO, orderDAO, hotelDAO)
	orderService := service.NewOrderService(orderDAO, roomDAO, paymentDAO, hotelDAO, db)
	paymentService := service.NewPaymentService(paymentDAO, orderDAO, db, cfg.PaymentMode)
	adminService := service.NewAdminService(orderDAO, paymentDAO, roomDAO, hotelDAO, userDAO)

	// ── Handlers ──
	authH := handler.NewAuthHandler(authService, userDAO)
	hotelH := handler.NewHotelHandler(hotelService)
	roomH := handler.NewRoomHandler(roomService)
	orderH := handler.NewOrderHandler(orderService, orderDAO)
	paymentH := handler.NewPaymentHandler(paymentService, paymentDAO, orderDAO, db)
	adminH := handler.NewAdminHandler(adminService, roomService, userDAO)

	// ── Router ──
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())

	// ── Rate limiter: 60 req/min per IP on auth endpoints ──
	rateLimitAuth := middleware.RateLimit(rdb, 10, time.Minute) // 10 req/min for auth
	rateLimitAPI := middleware.RateLimit(rdb, 120, time.Minute) // 120 req/min for API

	// Health
	r.GET("/api/health", middleware.HealthCheck)

	// Root — 功能完整的单页 Web 界面
	r.GET("/", serveHomePage)

	// Root
	r.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "智慧酒店管理系统 API 运行中", "version": "2.0.0", "lang": "Go"})
	})

	// ==================== Public Routes ====================

	r.POST("/api/auth/send-code", rateLimitAuth, authH.SendCode)
	r.POST("/api/auth/register", rateLimitAuth, authH.Register)
	r.POST("/api/auth/login", rateLimitAuth, authH.Login)

	r.GET("/api/hotels", hotelH.List)
	r.GET("/api/hotels/:hotel_id", hotelH.Get)

	r.GET("/api/rooms", roomH.GetAvailableRooms)
	r.GET("/api/rooms/types", roomH.ListRoomTypes)
	r.GET("/api/rooms/calendar", roomH.GetCalendar)

	// Payment callbacks (external platform webhooks, no auth)
	r.POST("/api/payment/callback/wechat", paymentH.WechatCallback)
	r.POST("/api/payment/callback/alipay", paymentH.AlipayCallback)

	// ==================== Protected Routes (JWT required) ====================

	protected := r.Group("/api")
	protected.Use(middleware.JWTAuth(cfg, db, rdb))
	{
		// Auth
		protected.GET("/auth/me", authH.GetMe)
		protected.POST("/auth/logout", func(c *gin.Context) {
			tokenStr, _ := c.Get("token_str")
			if rdb != nil && tokenStr != nil {
				rdb.SetEx(c.Request.Context(), "jwt:blacklist:"+tokenStr.(string), "1",
					time.Duration(cfg.JWTExpireHours)*time.Hour)
			}
			c.JSON(200, gin.H{"code": 0, "message": "已退出登录"})
		})

		// Hotels (super admin only)
		protected.POST("/hotels", middleware.RequireSuperAdmin(), hotelH.Create)
		protected.PUT("/hotels/:hotel_id", middleware.RequireSuperAdmin(), hotelH.Update)

		// Room Types (admin)
		protected.POST("/rooms/types", middleware.RequireHotelAdmin(), roomH.CreateRoomType)
		protected.PUT("/rooms/types/:type_id", middleware.RequireHotelAdmin(), roomH.UpdateRoomType)

		// Rooms (admin)
		protected.GET("/rooms/all", middleware.RequireHotelAdmin(), roomH.GetAllRooms)
		protected.POST("/rooms", middleware.RequireHotelAdmin(), roomH.CreateRoom)
		protected.POST("/rooms/batch", middleware.RequireHotelAdmin(), roomH.BatchCreateRooms)
		protected.PUT("/rooms/:room_id", middleware.RequireHotelAdmin(), roomH.UpdateRoom)
		protected.DELETE("/rooms/:room_id", middleware.RequireHotelAdmin(), roomH.DeleteRoom)

		// Orders (authenticated users)
		protected.POST("/orders", rateLimitAPI, orderH.CreateOrder)
		protected.GET("/orders", orderH.List)
		protected.GET("/orders/:order_id", orderH.GetDetail)
		protected.PUT("/orders/:order_id/cancel", orderH.Cancel)
		protected.GET("/orders/:order_id/print", orderH.PrintOrder)

		// Orders (staff: check-in/out/extend)
		protected.PUT("/orders/:order_id/check-in", middleware.RequireHotelStaff(), orderH.CheckIn)
		protected.PUT("/orders/:order_id/check-out", middleware.RequireHotelStaff(), orderH.CheckOut)
		protected.PUT("/orders/:order_id/extend", middleware.RequireHotelStaff(), orderH.ExtendStay)

		// Payment
		protected.GET("/payment/methods", paymentH.Methods)
		protected.POST("/payment/create", paymentH.Create)
		protected.GET("/payment/status/:payment_id", paymentH.Status)

		// Admin
		protected.GET("/admin/dashboard", middleware.RequireHotelAdmin(), adminH.Dashboard)
		protected.GET("/admin/staff", middleware.RequireHotelAdmin(), adminH.ListStaff)
		protected.POST("/auth/admin/staff", middleware.RequireHotelAdmin(), authH.CreateStaff)
		protected.DELETE("/admin/staff/:user_id", middleware.RequireHotelAdmin(), adminH.DeleteStaff)
		protected.PUT("/admin/staff/:user_id/toggle", middleware.RequireHotelAdmin(), adminH.ToggleStaff)
		protected.GET("/auth/admin/login-logs", middleware.RequireHotelAdmin(), authH.GetLoginLogs)
		protected.GET("/admin/export", middleware.RequireHotelAdmin(), adminH.ExportOrders)
	}

	return r
}
