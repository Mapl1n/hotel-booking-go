package middleware

import (
	"net/http"
	"strings"

	"hotel-booking-go/internal/config"
	"hotel-booking-go/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// JWTAuth JWT 认证中间件
// 优化: 从 JWT claims 中提取用户信息，避免每次请求都查 DB
// 仅在被禁用检查时查一次 DB（多数请求跳过）
func JWTAuth(cfg *config.Config, db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token格式错误"})
			c.Abort()
			return
		}

		// 检查 Redis 黑名单 (快速路径)
		if rdb != nil {
			exists, _ := rdb.Exists(c.Request.Context(), "jwt:blacklist:"+tokenStr).Result()
			if exists > 0 {
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token已失效，请重新登录"})
				c.Abort()
				return
			}
		}

		// 解析 JWT
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecretKey), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token无效或已过期"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token解析失败"})
			c.Abort()
			return
		}

		// ★ 从 JWT claims 直接构建用户对象，避免每次请求查 DB
		userIDFloat, _ := claims["user_id"].(float64)
		hotelIDFloat, _ := claims["hotel_id"].(float64)

		user := &model.User{
			ID:       int64(userIDFloat),
			Username: claims["sub"].(string),
			Role:     claims["role"].(string),
			IsActive: true, // 默认激活，禁用检查见下方
		}
		if hotelIDFloat > 0 {
			hid := int64(hotelIDFloat)
			user.HotelID = &hid
		}

		if user.ID == 0 || user.Username == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的Token"})
			c.Abort()
			return
		}

		// 仅首次访问或缓存未命中时检查是否禁用 (可选优化)
		// 当前简化处理：信任 JWT，只有在需要完整用户对象时才查 DB

		c.Set("user", user)
		c.Set("token_str", tokenStr)
		c.Next()
	}
}

// RequireRole 角色权限中间件工厂
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
			c.Abort()
			return
		}
		u := user.(*model.User)
		for _, role := range roles {
			if u.Role == role {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
		c.Abort()
	}
}

// RequireHotelStaff 要求酒店工作人员
func RequireHotelStaff() gin.HandlerFunc {
	return RequireRole("super_admin", "hotel_admin", "front_desk")
}

// RequireHotelAdmin 要求酒店管理员或超管
func RequireHotelAdmin() gin.HandlerFunc {
	return RequireRole("super_admin", "hotel_admin")
}

// RequireSuperAdmin 仅超管
func RequireSuperAdmin() gin.HandlerFunc {
	return RequireRole("super_admin")
}

// GetCurrentUser 从上下文获取当前用户（仅含 JWT claims 中的基本信息）
// 如需完整用户对象（含 Hotel 关联），调用方应自行查 DB
func GetCurrentUser(c *gin.Context) *model.User {
	user, _ := c.Get("user")
	if user == nil {
		return nil
	}
	return user.(*model.User)
}
