package main

import "sync"

type safeMap[KT comparable, VT interface{}] struct {
	SafeMap[KT, VT]
	m              map[KT]VT
	defaultFactory func() VT
	sync.RWMutex
}

type SafeMap[KT comparable, VT interface{}] interface {
	Set(key KT, value VT)
	Ensure(key KT)
	Get(key KT) VT
	UnsafeGet(key KT) (VT, bool)
	Delete(key KT)
	Clear()
	Size() int
	Contains(key KT) bool
	ForEach(fn func(key KT, value VT))
	Iterator() map[KT]VT
}

func NewSafeMapOf[T SafeMap[KT, VT], KT comparable, VT interface{}](df ...func() VT) SafeMap[KT, VT] {
	if len(df) == 0 {
		return &safeMap[KT, VT]{
			m: make(map[KT]VT),
		}
	}
	return &safeMap[KT, VT]{
		m:              make(map[KT]VT),
		defaultFactory: df[0],
	}
}

func (m *safeMap[KT, VT]) Set(key KT, value VT) {
	m.Lock()
	m.m[key] = value
	m.Unlock()
}

func (m *safeMap[KT, VT]) Ensure(key KT) {
	m.Lock()
	if _, ok := m.m[key]; !ok {
		m.m[key] = m.defaultFactory()
	}
	m.Unlock()
}

func (m *safeMap[KT, VT]) Get(key KT) VT {
	m.RLock()
	value := m.m[key]
	m.RUnlock()
	return value
}

func (m *safeMap[KT, VT]) UnsafeGet(key KT) (VT, bool) {
	value, ok := m.m[key]
	return value, ok
}

func (m *safeMap[KT, VT]) Delete(key KT) {
	m.Lock()
	delete(m.m, key)
	m.Unlock()
}

func (m *safeMap[KT, VT]) Clear() {
	m.Lock()
	m.m = make(map[KT]VT)
	m.Unlock()
}

func (m *safeMap[KT, VT]) Size() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.m)
}

func (m *safeMap[KT, VT]) Contains(key KT) bool {
	m.RLock()
	_, ok := m.m[key]
	m.RUnlock()
	return ok
}

func (m *safeMap[KT, VT]) ForEach(fn func(key KT, value VT)) {
	m.RLock()
	for k, v := range m.m {
		fn(k, v)
	}
	m.RUnlock()
}

func (m *safeMap[KT, VT]) Iterator() map[KT]VT {
	return m.m
}
