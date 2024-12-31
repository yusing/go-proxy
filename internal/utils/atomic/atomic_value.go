package atomic

import (
	"encoding/json"
	"sync/atomic"
)

type Value[T any] struct {
	atomic.Value
}

func (a *Value[T]) Load() T {
	return a.Value.Load().(T)
}

func (a *Value[T]) Store(v T) {
	a.Value.Store(v)
}

func (a *Value[T]) Swap(v T) T {
	return a.Value.Swap(v).(T)
}

func (a *Value[T]) CompareAndSwap(oldV, newV T) bool {
	return a.Value.CompareAndSwap(oldV, newV)
}

func (a *Value[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Load())
}
