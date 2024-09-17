package functional

import (
	"context"
	"sync"

	"gopkg.in/yaml.v3"

	E "github.com/yusing/go-proxy/error"
)

type Map[KT comparable, VT any] struct {
	m       map[KT]VT
	defVals map[KT]VT
	sync.RWMutex
}

// NewMap creates a new Map with the given map as its initial values.
//
// Parameters:
// - dv: optional default values for the Map
//
// Return:
// - *Map[KT, VT]: a pointer to the newly created Map.
func NewMap[KT comparable, VT any](dv ...map[KT]VT) *Map[KT, VT] {
	return NewMapFrom(make(map[KT]VT), dv...)
}

// NewMapOf creates a new Map with the given map as its initial values.
//
// Type parameters:
// - M: type for the new map.
//
// Parameters:
// - dv: optional default values for the Map
//
// Return:
// - *Map[KT, VT]: a pointer to the newly created Map.
func NewMapOf[M Map[KT, VT], KT comparable, VT any](dv ...map[KT]VT) *Map[KT, VT] {
	return NewMapFrom(make(map[KT]VT), dv...)
}

// NewMapFrom creates a new Map with the given map as its initial values.
//
// Parameters:
// - from: a map of type KT to VT, which will be the initial values of the Map.
// - dv: optional default values for the Map
//
// Return:
// - *Map[KT, VT]: a pointer to the newly created Map.
func NewMapFrom[KT comparable, VT any](from map[KT]VT, dv ...map[KT]VT) *Map[KT, VT] {
	if len(dv) > 0 {
		return &Map[KT, VT]{m: from, defVals: dv[0]}
	}
	return &Map[KT, VT]{m: from}
}

func (m *Map[KT, VT]) Set(key KT, value VT) {
	m.Lock()
	m.m[key] = value
	m.Unlock()
}

func (m *Map[KT, VT]) Get(key KT) VT {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.m[key]
	if !ok && m.defVals != nil {
		return m.defVals[key]
	}
	return value
}

// Find searches for the first element in the map that satisfies the given criteria.
//
// Parameters:
// - criteria: a function that takes a value of type VT and returns a tuple of any type and a boolean.
//
// Return:
// - any: the first value that satisfies the criteria, or nil if no match is found.
func (m *Map[KT, VT]) Find(criteria func(VT) (any, bool)) any {
	m.RLock()
	defer m.RUnlock()

	result := make(chan any)
	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, v := range m.m {
		wg.Add(1)
		go func(val VT) {
			defer wg.Done()
			if value, ok := criteria(val); ok {
				select {
				case result <- value:
					cancel() // Cancel other goroutines if a result is found
				case <-ctx.Done(): // If already cancelled
					return
				}
			}
		}(v)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	// The first valid match, if any
	select {
	case res, ok := <-result:
		if ok {
			return res
		}
	case <-ctx.Done():
	}

	return nil // Return nil if no matches found
}

func (m *Map[KT, VT]) UnsafeGet(key KT) (VT, bool) {
	value, ok := m.m[key]
	return value, ok
}

func (m *Map[KT, VT]) UnsafeSet(key KT, value VT) {
	m.m[key] = value
}

func (m *Map[KT, VT]) Delete(key KT) {
	m.Lock()
	delete(m.m, key)
	m.Unlock()
}

func (m *Map[KT, VT]) UnsafeDelete(key KT) {
	delete(m.m, key)
}

// MergeWith merges the contents of another Map[KT, VT]
// into the current Map[KT, VT] and
// returns a map that were duplicated.
//
// Parameters:
// - other: a pointer to another Map[KT, VT] to be merged into the current Map[KT, VT].
//
// Return:
// - Map[KT, VT]: a map of key-value pairs that were duplicated during the merge.
func (m *Map[KT, VT]) MergeWith(other *Map[KT, VT]) Map[KT, VT] {
	dups := make(map[KT]VT)

	m.Lock()
	for k, v := range other.m {
		if _, isDup := m.m[k]; !isDup {
			m.m[k] = v
		} else {
			dups[k] = v
		}
	}
	m.Unlock()
	return Map[KT, VT]{m: dups}
}

func (m *Map[KT, VT]) Clear() {
	m.Lock()
	m.m = make(map[KT]VT)
	m.Unlock()
}

func (m *Map[KT, VT]) Size() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.m)
}

func (m *Map[KT, VT]) Contains(key KT) bool {
	m.RLock()
	_, ok := m.m[key]
	m.RUnlock()
	return ok
}

func (m *Map[KT, VT]) Clone() *Map[KT, VT] {
	m.RLock()
	defer m.RUnlock()
	clone := make(map[KT]VT, len(m.m))
	for k, v := range m.m {
		clone[k] = v
	}
	return &Map[KT, VT]{m: clone, defVals: m.defVals}
}

func (m *Map[KT, VT]) EachKV(fn func(k KT, v VT)) {
	m.Lock()
	for k, v := range m.m {
		fn(k, v)
	}
	m.Unlock()
}

func (m *Map[KT, VT]) Each(fn func(v VT)) {
	m.Lock()
	for _, v := range m.m {
		fn(v)
	}
	m.Unlock()
}

func (m *Map[KT, VT]) EachParallel(fn func(v VT)) {
	m.Lock()
	ParallelForEachValue(m.m, fn)
	m.Unlock()
}

func (m *Map[KT, VT]) EachKVParallel(fn func(k KT, v VT)) {
	m.Lock()
	ParallelForEachKV(m.m, fn)
	m.Unlock()
}

func (m *Map[KT, VT]) UnmarshalFromYAML(data []byte) E.NestedError {
	return E.From(yaml.Unmarshal(data, m.m))
}

func (m *Map[KT, VT]) Iterator() map[KT]VT {
	return m.m
}
