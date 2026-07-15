package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit Redis 滑动窗口限流中间件
// maxRequests: 窗口内允许的最大请求数
// window: 时间窗口大小
func RateLimit(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		ip := c.ClientIP()
		key := "ratelimit:" + ip + ":" + c.FullPath()
		ctx := c.Request.Context()

		// Pipeline: INCR + EXPIRE
		pipe := rdb.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window)
		_, err := pipe.Exec(ctx)
		if err != nil {
			c.Next()
			return
		}

		if incr.Val() > int64(maxRequests) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
