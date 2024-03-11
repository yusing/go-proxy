package main

import "sync"

type safeMap[KT comparable, VT interface{}] struct {
	SafeMap[KT, VT]
	m              map[KT]VT
	mutex          sync.Mutex
	defaultFactory func() VT
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

func NewSafeMap[KT comparable, VT interface{}](df ...func() VT) SafeMap[KT, VT] {
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
	m.mutex.Lock()
	m.m[key] = value
	m.mutex.Unlock()
}

func (m *safeMap[KT, VT]) Ensure(key KT) {
	m.mutex.Lock()
	if _, ok := m.m[key]; !ok {
		m.m[key] = m.defaultFactory()
	}
	m.mutex.Unlock()
}

func (m *safeMap[KT, VT]) Get(key KT) VT {
	m.mutex.Lock()
	value := m.m[key]
	m.mutex.Unlock()
	return value
}

func (m *safeMap[KT, VT]) UnsafeGet(key KT) (VT, bool) {
	value, ok := m.m[key]
	return value, ok
}

func (m *safeMap[KT, VT]) Delete(key KT) {
	m.mutex.Lock()
	delete(m.m, key)
	m.mutex.Unlock()
}

func (m *safeMap[KT, VT]) Clear() {
	m.mutex.Lock()
	m.m = make(map[KT]VT)
	m.mutex.Unlock()
}

func (m *safeMap[KT, VT]) Size() int {
	m.mutex.Lock()
	size := len(m.m)
	m.mutex.Unlock()
	return size
}

func (m *safeMap[KT, VT]) Contains(key KT) bool {
	m.mutex.Lock()
	_, ok := m.m[key]
	m.mutex.Unlock()
	return ok
}

func (m *safeMap[KT, VT]) ForEach(fn func(key KT, value VT)) {
	m.mutex.Lock()
	for k, v := range m.m {
		fn(k, v)
	}
	m.mutex.Unlock()
}

func (m *safeMap[KT, VT]) Iterator() map[KT]VT {
	return m.m
}
