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
	Set(taskName, handleName string, jobId, expireAt int64)
	Get(taskName string) *Storage
	Del(taskName string)
	Exists(taskName string) bool
	Len() int
	GetAll() map[string]*Storage
}

type Storage struct {
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
	data map[string]*Storage
}

func NewSessionStorage() *SessionStorage {
	return &SessionStorage{data: make(map[string]*Storage)}
}

func (s *SessionStorage) Set(taskName, handleName string, jobId, expireAt int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[taskName] = &Storage{
		HandleName: handleName,
		ExpireAt:   expireAt,
		JobId:      jobId,
	}
}

func (s *SessionStorage) Get(taskName string) *Storage {
	s.mu.Lock()
	defer s.mu.Unlock()
	storage, exists := s.data[taskName]
	if !exists {
		return nil
	}

	return storage
}

func (s *SessionStorage) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *SessionStorage) GetAll() map[string]*Storage {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := make(map[string]*Storage, len(s.data))
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
	if !exists {
		return false
	}

	return true
}
