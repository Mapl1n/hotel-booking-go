package sms

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// memoryStore 内存存储（Redis 不可用时的降级方案）
type memoryStore struct {
	mu   sync.RWMutex
	data map[string]codeEntry
}

type codeEntry struct {
	code     string
	expireAt time.Time
}

var memStore = &memoryStore{data: make(map[string]codeEntry)}

func (m *memoryStore) set(key, code string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = codeEntry{code: code, expireAt: time.Now().Add(ttl)}
}

func (m *memoryStore) get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.data[key]
	if !ok || time.Now().After(entry.expireAt) {
		return "", false
	}
	return entry.code, true
}

func (m *memoryStore) del(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *memoryStore) ttl(key string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.data[key]
	if !ok {
		return -1
	}
	remaining := time.Until(entry.expireAt)
	if remaining < 0 {
		return -1
	}
	return remaining
}

// cleanExpired 清理过期条目
func (m *memoryStore) cleanExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, v := range m.data {
		if now.After(v.expireAt) {
			delete(m.data, k)
		}
	}
}

// Service SMS 服务，优先使用 Redis，降级到内存
type Service struct {
	rdb      *redis.Client
	provider string
	mockCode string
}

func NewService(rdb *redis.Client, provider, mockCode string) *Service {
	return &Service{rdb: rdb, provider: provider, mockCode: mockCode}
}

// SendCode 发送验证码
func (s *Service) SendCode(ctx context.Context, phone string) error {
	key := fmt.Sprintf("sms:code:%s", phone)
	memStore.cleanExpired()

	var remaining time.Duration

	if s.rdb != nil {
		ttl, err := s.rdb.TTL(ctx, key).Result()
		if err != nil {
			return err
		}
		remaining = ttl - 240*time.Second
	} else {
		remaining = memStore.ttl(key) - 240*time.Second
	}

	if remaining > 0 {
		return fmt.Errorf("请 %d 秒后再试", int(remaining.Seconds()))
	}

	code := s.mockCode
	if s.provider == "mock" {
		fmt.Printf("[SMS-MOCK] → %s  验证码: %s\n", phone, code)
	}

	if s.rdb != nil {
		return s.rdb.SetEx(ctx, key, code, 5*time.Minute).Err()
	}
	memStore.set(key, code, 5*time.Minute)
	return nil
}

// VerifyCode 校验验证码（一次性，验证后删除）
func (s *Service) VerifyCode(ctx context.Context, phone, code string) bool {
	key := fmt.Sprintf("sms:code:%s", phone)
	memStore.cleanExpired()

	if s.rdb != nil {
		stored, err := s.rdb.Get(ctx, key).Result()
		if err != nil {
			return false
		}
		if stored != code {
			return false
		}
		s.rdb.Del(ctx, key)
		return true
	}

	stored, ok := memStore.get(key)
	if !ok || stored != code {
		return false
	}
	memStore.del(key)
	return true
}
