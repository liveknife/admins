package middlewares

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-demo/database"

	"github.com/gin-gonic/gin"
)

// RateLimit 按 (客户端 IP + 路径 + 可选 key) 做滑动窗口限流。
// 优先使用 Redis 计数；Redis 不可用时降级到进程内 map（多副本部署下降级不严谨，仅保底）。
// - max: 窗口内允许的最大次数
// - window: 时间窗口大小
// - keyFn: 从 gin.Context 提取额外维度（例如 email、captcha_id）；返回 "" 表示只按 IP+路径
func RateLimit(name string, max int, window time.Duration, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		extra := ""
		if keyFn != nil {
			extra = keyFn(c)
		}
		key := "rl:" + name + ":" + c.ClientIP() + ":" + c.FullPath()
		if extra != "" {
			key += ":" + shortHash(extra)
		}
		if count, ok := incrementCounter(c.Request.Context(), key, window); ok && count > int64(max) {
			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please retry later"})
			return
		}
		c.Next()
	}
}

func shortHash(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:6])
}

// incrementCounter 返回窗口内的当前计数；返回 (0, false) 表示计数不可用（放行不拦截）
func incrementCounter(ctx context.Context, key string, window time.Duration) (int64, bool) {
	if database.RedisClient != nil {
		reqCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		pipe := database.RedisClient.TxPipeline()
		incr := pipe.Incr(reqCtx, key)
		pipe.Expire(reqCtx, key, window)
		if _, err := pipe.Exec(reqCtx); err == nil {
			return incr.Val(), true
		}
	}
	return memoryCounter.hit(key, window), true
}

// ──────────────────────────────────────────────
// 进程内保底计数器（Redis 掉线时）
// ──────────────────────────────────────────────

type memCounter struct {
	mu   sync.Mutex
	buck map[string]memEntry
}

type memEntry struct {
	count     int64
	expiresAt time.Time
}

var memoryCounter = &memCounter{buck: map[string]memEntry{}}

func (m *memCounter) hit(key string, window time.Duration) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	// 顺手清理，避免无限增长
	if len(m.buck) > 4096 {
		for k, v := range m.buck {
			if v.expiresAt.Before(now) {
				delete(m.buck, k)
			}
		}
	}
	entry, ok := m.buck[key]
	if !ok || entry.expiresAt.Before(now) {
		entry = memEntry{count: 0, expiresAt: now.Add(window)}
	}
	entry.count++
	m.buck[key] = entry
	return entry.count
}
