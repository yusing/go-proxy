package functional

type Slice[T any] struct {
	s []T
}

func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{make([]T, 0)}
}

func NewSliceN[T any](n int) *Slice[T] {
	return &Slice[T]{make([]T, n)}
}

func NewSliceFrom[T any](s []T) *Slice[T] {
	return &Slice[T]{s}
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
	return &Slice[T]{append(s.s, e)}
}

func (s *Slice[T]) AddRange(other *Slice[T]) *Slice[T] {
	return &Slice[T]{append(s.s, other.s...)}
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
	return &Slice[T]{n}
}

func (s *Slice[T]) Filter(f func(T) bool) *Slice[T] {
	n := make([]T, 0)
	for _, v := range s.s {
		if f(v) {
			n = append(n, v)
		}
	}
	return &Slice[T]{n}
}
