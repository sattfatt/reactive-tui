package signal

import "sync"

// Global re-render hook — set by the app to trigger a full re-render.
var OnChange func()

var (
	batchDepth int
	batchMu    sync.Mutex
	dirty      bool
)

// Batch defers re-render notifications until the function completes.
func Batch(fn func()) {
	batchMu.Lock()
	batchDepth++
	batchMu.Unlock()

	fn()

	batchMu.Lock()
	batchDepth--
	flush := batchDepth == 0 && dirty
	if flush {
		dirty = false
	}
	batchMu.Unlock()

	if flush {
		notify()
	}
}

func notify() {
	if OnChange != nil {
		batchMu.Lock()
		if batchDepth > 0 {
			dirty = true
			batchMu.Unlock()
			return
		}
		batchMu.Unlock()
		OnChange()
	}
}

// Signal holds a reactive value. When Set is called with a new value,
// all watchers are notified and a re-render is triggered.
type Signal[T comparable] struct {
	value    T
	watchers []func(T)
	mu       sync.RWMutex
}

func New[T comparable](initial T) *Signal[T] {
	return &Signal[T]{value: initial}
}

func (s *Signal[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	if s.value == v {
		s.mu.Unlock()
		return
	}
	s.value = v
	watchers := make([]func(T), len(s.watchers))
	copy(watchers, s.watchers)
	s.mu.Unlock()

	for _, w := range watchers {
		w(v)
	}
	notify()
}

// Update applies a function to the current value and sets the result.
func (s *Signal[T]) Update(fn func(T) T) {
	s.Set(fn(s.Get()))
}

func (s *Signal[T]) Watch(fn func(T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watchers = append(s.watchers, fn)
}

// ListSignal holds a reactive slice. Since slices aren't comparable,
// it always triggers on Set.
type ListSignal[T any] struct {
	value    []T
	watchers []func([]T)
	mu       sync.RWMutex
}

func NewList[T any](initial ...T) *ListSignal[T] {
	return &ListSignal[T]{value: initial}
}

func (s *ListSignal[T]) Get() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]T, len(s.value))
	copy(out, s.value)
	return out
}

func (s *ListSignal[T]) Set(v []T) {
	s.mu.Lock()
	s.value = v
	watchers := make([]func([]T), len(s.watchers))
	copy(watchers, s.watchers)
	s.mu.Unlock()

	for _, w := range watchers {
		w(v)
	}
	notify()
}

func (s *ListSignal[T]) Append(items ...T) {
	s.mu.Lock()
	s.value = append(s.value, items...)
	val := make([]T, len(s.value))
	copy(val, s.value)
	watchers := make([]func([]T), len(s.watchers))
	copy(watchers, s.watchers)
	s.mu.Unlock()

	for _, w := range watchers {
		w(val)
	}
	notify()
}

func (s *ListSignal[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.value)
}

func (s *ListSignal[T]) Watch(fn func([]T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watchers = append(s.watchers, fn)
}
