package utils

import "sync"

type SyncMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		mu:   sync.RWMutex{},
		data: map[K]V{},
	}
}

func (m *SyncMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.data)
}

func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok = m.data[key]

	return value, ok
}

func (m *SyncMap[K, V]) Swap(key K, value V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldValue, exist := m.data[key]
	m.data[key] = value

	return oldValue, exist
}

func (m *SyncMap[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *SyncMap[K, V]) Stores(data map[K]V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range data {
		m.data[k] = v
	}
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *SyncMap[K, V]) KeySlice() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slice := make([]K, 0, len(m.data))
	for k := range m.data {
		slice = append(slice, k)
	}

	return slice
}

func (m *SyncMap[K, V]) ValueSlice() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slice := make([]V, 0, len(m.data))
	for _, v := range m.data {
		slice = append(slice, v)
	}

	return slice
}

func (m *SyncMap[K, V]) Do(key K, fn func(V)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		fn(m.data[key])
	}
}

func (m *SyncMap[K, V]) Exec(fn func(map[K]V)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(m.data)
}
