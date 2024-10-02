package functional

import (
	"encoding/json"
	"sync"
)

type Slice[T any] struct {
	s  []T
	mu sync.Mutex
}

func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{s: make([]T, 0)}
}

func NewSliceN[T any](n int) *Slice[T] {
	return &Slice[T]{s: make([]T, n)}
}

func NewSliceFrom[T any](s []T) *Slice[T] {
	return &Slice[T]{s: s}
}

func (s *Slice[T]) Size() int {
	return len(s.s)
}

func (s *Slice[T]) Empty() bool {
	return len(s.s) == 0
}

func (s *Slice[T]) NotEmpty() bool {
	return len(s.s) > 0
}

func (s *Slice[T]) Iterator() []T {
	return s.s
}

func (s *Slice[T]) Set(i int, v T) {
	s.s[i] = v
}

func (s *Slice[T]) Add(e T) *Slice[T] {
	s.s = append(s.s, e)
	return s
}

func (s *Slice[T]) AddRange(other *Slice[T]) *Slice[T] {
	s.s = append(s.s, other.s...)
	return s
}

func (s *Slice[T]) SafeAdd(e T) *Slice[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Add(e)
}

func (s *Slice[T]) SafeAddRange(other *Slice[T]) *Slice[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.AddRange(other)
}

func (s *Slice[T]) Pop() T {
	v := s.s[len(s.s)-1]
	s.s = s.s[:len(s.s)-1]
	return v
}

func (s *Slice[T]) SafePop() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Pop()
}

func (s *Slice[T]) ForEach(do func(T)) {
	for _, v := range s.s {
		do(v)
	}
}

func (s *Slice[T]) Map(m func(T) T) *Slice[T] {
	n := make([]T, len(s.s))
	for i, v := range s.s {
		n[i] = m(v)
	}
	return &Slice[T]{s: n}
}

func (s *Slice[T]) Filter(f func(T) bool) *Slice[T] {
	n := make([]T, 0)
	for _, v := range s.s {
		if f(v) {
			n = append(n, v)
		}
	}
	return &Slice[T]{s: n}
}

func (s *Slice[T]) String() string {
	out, err := json.MarshalIndent(s.s, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(out)
}
