// 限流中间件
// 支持IP限流和令牌桶算法

package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	rate       float64   // 每秒产生令牌数
	capacity   int64     // 桶容量
	tokens     int64     // 当前令牌数
	lastUpdate time.Time // 上次更新时间
	mu         sync.Mutex
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(rate float64, capacity int64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastUpdate: time.Now(),
	}
}

// Allow 判断是否允许请求
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.lastUpdate = now

	// 添加新令牌
	tb.tokens += int64(elapsed * tb.rate)
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	// 判断是否有足够令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// IPRateLimiter IP限流器
type IPRateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // 每分钟请求数
	burst    int           // 突发容量
	cleanup  time.Duration // 清理周期
}

// Visitor 访问者
type Visitor struct {
	limiter  *TokenBucket
	lastSeen time.Time
}

// NewIPRateLimiter 创建IP限流器
func NewIPRateLimiter(rate int, burst int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  time.Minute,
	}

	// 启动清理协程
	go limiter.cleanupVisitors()

	return limiter
}

// GetVisitor 获取访问者限流器
func (l *IPRateLimiter) GetVisitor(ip string) *TokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()

	visitor, exists := l.visitors[ip]
	if !exists {
		visitor = &Visitor{
			limiter:  NewTokenBucket(float64(l.rate)/60.0, int64(l.burst)), // 转换为每秒
			lastSeen: time.Now(),
		}
		l.visitors[ip] = visitor
	}

	visitor.lastSeen = time.Now()
	return visitor.limiter
}

// cleanupVisitors 清理过期访问者
func (l *IPRateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		for ip, visitor := range l.visitors {
			if time.Since(visitor.lastSeen) > 10*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit IP限流中间件
func RateLimit(requestsPerMinute int, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(requestsPerMinute, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucket := limiter.GetVisitor(ip)

		if !bucket.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429001,
				"message": "too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// APILimit API特定限流
func APILimit(requestsPerSecond float64, capacity int64) gin.HandlerFunc {
	limiter := NewTokenBucket(requestsPerSecond, capacity)

	return func(c *gin.Context) {
		if !limiter.Allow("api") {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429002,
				"message": "api rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}