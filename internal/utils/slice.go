package utils

import (
	"sort"
	"sync"
)

type SyncSlice[T any] struct {
	mu   sync.RWMutex
	data []T
}

func NewSyncSlice[T any]() *SyncSlice[T] {
	return &SyncSlice[T]{
		mu:   sync.RWMutex{},
		data: []T{},
	}
}

func (s *SyncSlice[T]) Load(index int) (value T, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.data) {
		return value, false
	}

	return s.data[index], true
}

func (s *SyncSlice[T]) Store(index int, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.data) {
		return
	}

	s.data[index] = value
}

func (s *SyncSlice[T]) Delete(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.data) {
		return
	}

	s.data = append(s.data[:index], s.data[index+1:]...)
}

func (s *SyncSlice[T]) Append(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data, value)
}

func (s *SyncSlice[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data)
}

func (s *SyncSlice[T]) Sort(alg func(i, j int) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sort.Slice(s.data, alg)
}

func (s *SyncSlice[T]) Last() T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data[len(s.data)-1]
}
