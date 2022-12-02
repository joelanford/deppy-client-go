package util

import "sync"

func NewSyncMap[K comparable, V any]() SyncMap[K, V] {
	return SyncMap[K, V]{m: map[K]V{}}
}

type SyncMap[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

func (m *SyncMap[K, V]) Get(k K) V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.m[k]
}

func (m *SyncMap[K, V]) GetCheck(k K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.m[k]
	return v, ok
}

func (m *SyncMap[K, V]) Set(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[k] = v
}

func (m *SyncMap[K, V]) Delete(k K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
}

func (m *SyncMap[K, V]) Keys() []K {
	keys := make([]K, 0, len(m.m))
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k := range m.m {
		keys = append(keys, k)
	}
	return keys
}

func (m *SyncMap[K, V]) Values() []V {
	values := make([]V, 0, len(m.m))
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.m {
		values = append(values, v)
	}
	return values
}
