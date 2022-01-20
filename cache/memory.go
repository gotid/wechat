package cache

import (
	"sync"
	"time"
)

// Memory 提供一个基于内存的缓存。
type Memory struct {
	sync.Mutex

	data map[string]*data
}

var _ Cache = (*Memory)(nil)

type data struct {
	Data    interface{}
	Expired time.Time
}

// NewMemory 返回一个新的内存缓存。
func NewMemory() *Memory {
	return &Memory{
		data: map[string]*data{},
	}
}

func (m *Memory) Get(key string) interface{} {
	if v, ok := m.data[key]; ok {
		if v.Expired.Before(time.Now()) {
			m.deleteKey(key)
			return nil
		}
		return v.Data
	}
	return nil
}

func (m *Memory) Set(key string, val interface{}, timeout time.Duration) error {
	m.Lock()
	defer m.Unlock()

	m.data[key] = &data{
		Data:    val,
		Expired: time.Now().Add(timeout),
	}
	return nil
}

func (m *Memory) Exists(key string) bool {
	if v, ok := m.data[key]; ok {
		if v.Expired.Before(time.Now()) {
			m.deleteKey(key)
			return false
		}
		return true
	}
	return false
}

func (m *Memory) Delete(key string) error {
	m.deleteKey(key)
	return nil
}

func (m *Memory) deleteKey(key string) {
	m.Lock()
	defer m.Unlock()
	delete(m.data, key)
}
