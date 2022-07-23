package xxl

import "sync"

type Storage interface {
	Set(key, value string)
	Get(key string) string
	Del(key string)
	Exists(key string) bool
}

type SessionStorage struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewSessionStorage() *SessionStorage {
	return &SessionStorage{data: make(map[string]string)}
}

func (s *SessionStorage) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *SessionStorage) Get(key string) string {
	value, _ := s.data[key]
	return value
}

func (s *SessionStorage) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *SessionStorage) Exists(key string) bool {
	_, exists := s.data[key]
	return exists
}
