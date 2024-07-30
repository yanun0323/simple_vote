package utils

import "sync"

type SyncValue[T any] struct {
	mu    sync.RWMutex
	value T
}

func NewSyncValue[T any](value T) *SyncValue[T] {
	return &SyncValue[T]{
		mu:    sync.RWMutex{},
		value: value,
	}
}

func (v *SyncValue[T]) Load() (value T) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.value
}

func (v *SyncValue[T]) Store(value T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.value = value
}

func (v *SyncValue[T]) Delete() {
	v.mu.Lock()
	defer v.mu.Unlock()
	var zero T
	v.value = zero
}

func (v *SyncValue[T]) Swap(value T) T {
	v.mu.Lock()
	defer v.mu.Unlock()
	old := v.value
	v.value = value

	return old
}
