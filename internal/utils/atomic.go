package utils

import (
	"encoding/json"
	"sync/atomic"
)

type AtomicValue[T any] struct {
	atomic.Value
}

func (a *AtomicValue[T]) Load() T {
	return a.Value.Load().(T)
}

func (a *AtomicValue[T]) Store(v T) {
	a.Value.Store(v)
}

func (a *AtomicValue[T]) Swap(v T) T {
	return a.Value.Swap(v).(T)
}

func (a *AtomicValue[T]) CompareAndSwap(oldV, newV T) bool {
	return a.Value.CompareAndSwap(oldV, newV)
}

func (a *AtomicValue[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Load())
}
