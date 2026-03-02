package signal

import (
	"sync"
	"testing"
)

func resetGlobals() {
	batchMu.Lock()
	batchDepth = 0
	dirty = false
	batchMu.Unlock()
	OnChange = nil
}

func TestNew(t *testing.T) {
	s := New(42)
	if s.Get() != 42 {
		t.Errorf("expected 42, got %d", s.Get())
	}
}

func TestSetGet(t *testing.T) {
	s := New(0)
	s.Set(10)
	if s.Get() != 10 {
		t.Errorf("expected 10, got %d", s.Get())
	}
}

func TestSetSameValueNoNotify(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	s := New(5)
	s.Set(5) // same value
	if count != 0 {
		t.Errorf("expected 0 notifications, got %d", count)
	}
}

func TestSetDifferentValueNotifies(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	s := New(5)
	s.Set(10)
	if count != 1 {
		t.Errorf("expected 1 notification, got %d", count)
	}
}

func TestWatch(t *testing.T) {
	resetGlobals()
	s := New(0)
	var received []int
	s.Watch(func(v int) {
		received = append(received, v)
	})

	s.Set(1)
	s.Set(2)
	s.Set(2) // same, should not trigger

	if len(received) != 2 {
		t.Errorf("expected 2 watch calls, got %d", len(received))
	}
	if received[0] != 1 || received[1] != 2 {
		t.Errorf("unexpected values: %v", received)
	}
}

func TestMultipleWatchers(t *testing.T) {
	resetGlobals()
	s := New(0)
	var mu sync.Mutex
	calls1, calls2 := 0, 0

	s.Watch(func(int) {
		mu.Lock()
		calls1++
		mu.Unlock()
	})
	s.Watch(func(int) {
		mu.Lock()
		calls2++
		mu.Unlock()
	})

	s.Set(1)

	mu.Lock()
	defer mu.Unlock()
	if calls1 != 1 || calls2 != 1 {
		t.Errorf("expected both watchers called once, got %d and %d", calls1, calls2)
	}
}

func TestUpdate(t *testing.T) {
	resetGlobals()
	s := New(10)
	s.Update(func(v int) int { return v + 5 })
	if s.Get() != 15 {
		t.Errorf("expected 15, got %d", s.Get())
	}
}

func TestBatch(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	s1 := New(0)
	s2 := New(0)

	Batch(func() {
		s1.Set(1)
		s2.Set(2)
	})

	if count != 1 {
		t.Errorf("expected 1 notification from batch, got %d", count)
	}
	if s1.Get() != 1 || s2.Get() != 2 {
		t.Errorf("values not set correctly: s1=%d s2=%d", s1.Get(), s2.Get())
	}
}

func TestBatchNested(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	s := New(0)

	Batch(func() {
		s.Set(1)
		Batch(func() {
			s.Set(2)
		})
		s.Set(3)
	})

	if count != 1 {
		t.Errorf("expected 1 notification from nested batch, got %d", count)
	}
	if s.Get() != 3 {
		t.Errorf("expected 3, got %d", s.Get())
	}
}

func TestBatchNoChanges(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	Batch(func() {
		// no signal changes
	})

	if count != 0 {
		t.Errorf("expected 0 notifications, got %d", count)
	}
}

func TestSignalString(t *testing.T) {
	s := New("hello")
	if s.Get() != "hello" {
		t.Errorf("expected hello, got %s", s.Get())
	}
	s.Set("world")
	if s.Get() != "world" {
		t.Errorf("expected world, got %s", s.Get())
	}
}

// ListSignal tests

func TestNewList(t *testing.T) {
	l := NewList("a", "b", "c")
	got := l.Get()
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("unexpected list: %v", got)
	}
}

func TestNewListEmpty(t *testing.T) {
	l := NewList[string]()
	if l.Len() != 0 {
		t.Errorf("expected empty list, got len %d", l.Len())
	}
}

func TestListSet(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	l := NewList[int]()
	l.Set([]int{1, 2, 3})
	if l.Len() != 3 {
		t.Errorf("expected len 3, got %d", l.Len())
	}
	if count != 1 {
		t.Errorf("expected 1 notification, got %d", count)
	}
}

func TestListAppend(t *testing.T) {
	resetGlobals()
	count := 0
	OnChange = func() { count++ }

	l := NewList(1, 2)
	l.Append(3, 4)
	got := l.Get()
	if len(got) != 4 {
		t.Errorf("expected 4 items, got %d", len(got))
	}
	if got[2] != 3 || got[3] != 4 {
		t.Errorf("unexpected values: %v", got)
	}
	if count != 1 {
		t.Errorf("expected 1 notification, got %d", count)
	}
}

func TestListWatch(t *testing.T) {
	resetGlobals()
	l := NewList[string]()
	var received [][]string
	l.Watch(func(v []string) {
		received = append(received, v)
	})

	l.Set([]string{"a"})
	l.Append("b")

	if len(received) != 2 {
		t.Errorf("expected 2 watch calls, got %d", len(received))
	}
}

func TestListGetReturnsCopy(t *testing.T) {
	l := NewList(1, 2, 3)
	got := l.Get()
	got[0] = 999
	original := l.Get()
	if original[0] != 1 {
		t.Error("Get() should return a copy, not a reference")
	}
}

func TestSignalConcurrency(t *testing.T) {
	resetGlobals()
	s := New(0)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			s.Set(v)
			_ = s.Get()
		}(i)
	}

	wg.Wait()
	// Just verify no panic/race
}
