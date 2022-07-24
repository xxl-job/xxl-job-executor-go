package xxl

import (
	"sync"
	"time"
)

const (
	Persistence = -1
)

type Storager interface {
	// Set expireAt 过期时间, 时间戳
	Set(key, value string, expireAt int64)
	Get(key string) string
	Del(key string)
	Exists(key string) bool
	Len() int
	GetAll() map[string]Storage
}

type Storage struct {
	TaskName   string
	HandleName string
	ExpireAt   int64
}

// Expired 已过期
func (j *Storage) Expired() bool {
	if j == nil {
		return true
	}

	if j.ExpireAt == Persistence {
		return false
	}

	expire := time.Now().Unix() >= j.ExpireAt
	return expire
}

// Persistence 是否为永久
func (j *Storage) Persistence() bool {
	if j == nil {
		return false
	}

	return j.ExpireAt == Persistence
}

type SessionStorage struct {
	mu   sync.RWMutex
	data map[string]Storage
}

func NewSessionStorage() *SessionStorage {
	return &SessionStorage{data: make(map[string]Storage)}
}

func (s *SessionStorage) Set(key, value string, expireAt int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = Storage{
		TaskName:   key,
		HandleName: value,
		ExpireAt:   expireAt,
	}
}

func (s *SessionStorage) Get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	storage, exists := s.data[key]
	if !exists {
		return ""
	}

	if storage.Expired() {
		delete(s.data, key)
		return ""
	}
	return storage.HandleName
}

func (s *SessionStorage) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *SessionStorage) GetAll() map[string]Storage {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := make(map[string]Storage, len(s.data))
	for k, v := range s.data {
		data[k] = v
	}
	return data
}

func (s *SessionStorage) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}

func (s *SessionStorage) Exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.data[key]
	return exists
}
