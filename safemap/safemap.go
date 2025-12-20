package safemap

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
)

type SafeMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func New[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

func (s *SafeMap[K, V]) Set(k K, v V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
}

func (s *SafeMap[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[k]
	return val, ok
}

func (s *SafeMap[K, V]) Delete(k K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, k)
}

func (s *SafeMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
func (s *SafeMap[K, V]) SortedForEach(f func(K, V) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect keys and sort them
	keys := make([]K, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})

	// Iterate over sorted keys
	for _, k := range keys {
		if f(k, s.data[k]) {
			break
		}
	}

}
func (s *SafeMap[K, V]) ForEach(f func(K, V) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, val := range s.data {
		stop := f(key, val)
		if stop {
			break
		}
	}
}

func (s *SafeMap[K, V]) Remove(f func(K, V) bool) map[K]V {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := make(map[K]V)
	for key, val := range s.data {
		remove := f(key, val)
		if remove {
			delete(s.data, key)
			r[key] = val
		}
	}
	return r
}

func (s *SafeMap[K, V]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[K]V)
}

func (s *SafeMap[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r := make([]K, 0)
	for key := range s.data {
		r = append(r, key)
	}
	return r
}

func (s *SafeMap[K, V]) Values() []V {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r := make([]V, 0)
	for _, val := range s.data {
		r = append(r, val)
	}
	return r
}

func (s *SafeMap[K, V]) Random() (int, V) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.data) == 0 {
		var zeroV V
		return 0, zeroV
	}

	keys := make([]K, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}

	idx := rand.Intn(len(keys))
	return idx, s.data[keys[idx]]
}
