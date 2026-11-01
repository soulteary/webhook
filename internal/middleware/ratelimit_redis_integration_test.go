package middleware

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Exercise the application's legacy connection configuration and shared quota
// against Redis, rather than relying on the in-process Lua emulator alone.
func TestRedisRateLimitCompatibility(t *testing.T) {
	addr := os.Getenv("WEBHOOK_REDIS_TEST_ADDR")
	if addr == "" {
		if os.Getenv("WEBHOOK_REDIS_TEST_REQUIRED") == "1" {
			t.Fatal("WEBHOOK_REDIS_TEST_ADDR is required")
		}
		t.Skip("set WEBHOOK_REDIS_TEST_ADDR for the real Redis compatibility check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	password := os.Getenv("WEBHOOK_REDIS_TEST_PASSWORD")
	admin := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: 9})
	defer admin.Close()
	if err := admin.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("webhook:kit-compat:%d:", time.Now().UnixNano())
	defer admin.Del(context.Background(), prefix+"quota", prefix+"isolated")
	config := RateLimitConfig{Enabled: true, RPS: 3, Burst: 3, RedisEnabled: true, RedisAddr: addr, RedisPassword: password, RedisDB: 9, RedisKeyPrefix: prefix, WindowSeconds: 60}
	first, second := newRedisCompatibilityLimiter(t, config), newRedisCompatibilityLimiter(t, config)
	for i, limiter := range []*RateLimiter{first, second, first} {
		allowed, remaining, retryAfter := limiter.checkRedisLimit(ctx, "quota", 3)
		assertRedisQuota(t, allowed, remaining, retryAfter, true, 2-i)
	}
	allowed, remaining, retryAfter := second.checkRedisLimit(ctx, "quota", 3)
	assertRedisQuota(t, allowed, remaining, retryAfter, false, 0)
	if allowed, _, _ := second.checkRedisLimit(ctx, "isolated", 3); !allowed {
		t.Fatal("one key exhausted another key's quota")
	}
	if ttl := admin.PTTL(ctx, prefix+"quota").Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("quota TTL = %v, want a bounded 60-second window", ttl)
	}
}

func newRedisCompatibilityLimiter(t *testing.T, config RateLimitConfig) *RateLimiter {
	t.Helper()
	limiter := NewRateLimiter(config)
	if limiter == nil || !limiter.IsRedisEnabled() {
		t.Fatal("real Redis setup silently fell back to in-memory limiting")
	}
	t.Cleanup(func() {
		if err := limiter.Close(); err != nil {
			t.Error(err)
		}
	})
	return limiter
}

func assertRedisQuota(t *testing.T, allowed bool, remaining int, retryAfter time.Duration, wantAllowed bool, wantRemaining int) {
	t.Helper()
	if allowed != wantAllowed || remaining != wantRemaining || retryAfter <= 0 || retryAfter > time.Minute {
		t.Fatalf("quota: allowed=%v remaining=%d retryAfter=%v; want allowed=%v remaining=%d and bounded expiry", allowed, remaining, retryAfter, wantAllowed, wantRemaining)
	}
}
