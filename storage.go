package xxl

import (
	"encoding/json"
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
	JobId      int64
}

// Equal 相等
func (s *Storage) Equal(storage *Storage) bool {
	marshal1, _ := json.Marshal(s)
	marshal2, _ := json.Marshal(storage)
	equal := md5V(marshal1) == md5V(marshal2)
	return equal
}

// Expired 已过期
func (s *Storage) Expired() bool {
	if s == nil {
		return true
	}

	if s.ExpireAt == Persistence {
		return false
	}

	expire := time.Now().Unix() >= s.ExpireAt
	return expire
}

// Persistence 是否为永久
func (s *Storage) Persistence() bool {
	if s == nil {
		return false
	}

	return s.ExpireAt == Persistence
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
	storage, exists := s.data[key]
	if !exists {
		return false
	}

	if storage.Expired() {
		return false
	}

	return true
}
