package functional

import (
	"errors"
	"sync"

	"github.com/puzpuzpuz/xsync/v3"
	"gopkg.in/yaml.v3"
)

type Map[KT comparable, VT any] struct {
	*xsync.MapOf[KT, VT]
}

const minParallelSize = 4

func NewMapOf[KT comparable, VT any](options ...func(*xsync.MapConfig)) Map[KT, VT] {
	return Map[KT, VT]{xsync.NewMapOf[KT, VT](options...)}
}

func NewMapFrom[KT comparable, VT any](m map[KT]VT) (res Map[KT, VT]) {
	res = NewMapOf[KT, VT](xsync.WithPresize(len(m)))
	for k, v := range m {
		res.Store(k, v)
	}
	return
}

func NewMap[MapType Map[KT, VT], KT comparable, VT any]() Map[KT, VT] {
	return NewMapOf[KT, VT]()
}

// MapFind iterates over the map and returns the first value
// that satisfies the given criteria. The iteration is stopped
// once a value is found. If no value satisfies the criteria,
// the function returns the zero value of CT.
//
// The criteria function takes a value of type VT and returns a
// value of type CT and a boolean indicating whether the value
// satisfies the criteria. The boolean value is used to determine
// whether the iteration should be stopped.
//
// The function is safe for concurrent use.
func MapFind[KT comparable, VT, CT any](m Map[KT, VT], criteria func(VT) (CT, bool)) (_ CT) {
	result := make(chan CT, 1)

	m.Range(func(key KT, value VT) bool {
		select {
		case <-result: // already have a result
			return false // stop iteration
		default:
			if got, ok := criteria(value); ok {
				result <- got
				return false
			}
			return true
		}
	})

	select {
	case v := <-result:
		return v
	default:
		return
	}
}

// MergeFrom merges the contents of another Map into this one, ignoring duplicated keys.
//
// Parameters:
//
//	other: Map of values to add from
//
// Returns:
//
//	Map of duplicated keys-value pairs
func (m Map[KT, VT]) MergeFrom(other Map[KT, VT]) Map[KT, VT] {
	dups := NewMapOf[KT, VT]()

	other.Range(func(k KT, v VT) bool {
		if _, ok := m.Load(k); ok {
			dups.Store(k, v)
		} else {
			m.Store(k, v)
		}
		return true
	})
	return dups
}

// RangeAll calls the given function for each key-value pair in the map.
//
// Parameters:
//
//	do: function to call for each key-value pair
//
// Returns:
//
//	nothing
func (m Map[KT, VT]) RangeAll(do func(k KT, v VT)) {
	m.Range(func(k KT, v VT) bool {
		do(k, v)
		return true
	})
}

// RangeAllParallel calls the given function for each key-value pair in the map,
// in parallel. The map is not safe for modification from within the function.
//
// Parameters:
//
//	do: function to call for each key-value pair
//
// Returns:
//
//	nothing
func (m Map[KT, VT]) RangeAllParallel(do func(k KT, v VT)) {
	if m.Size() < minParallelSize {
		m.RangeAll(do)
		return
	}

	var wg sync.WaitGroup

	m.Range(func(k KT, v VT) bool {
		wg.Add(1)
		go func() {
			do(k, v)
			wg.Done()
		}()
		return true
	})
	wg.Wait()
}

// CollectErrors calls the given function for each key-value pair in the map,
// then returns a slice of errors collected.
func (m Map[KT, VT]) CollectErrors(do func(k KT, v VT) error) []error {
	errs := make([]error, 0)
	m.Range(func(k KT, v VT) bool {
		if err := do(k, v); err != nil {
			errs = append(errs, err)
		}
		return true
	})
	return errs
}

// CollectErrors calls the given function for each key-value pair in the map,
// then returns a slice of errors collected.
func (m Map[KT, VT]) CollectErrorsParallel(do func(k KT, v VT) error) []error {
	if m.Size() < minParallelSize {
		return m.CollectErrors(do)
	}

	errs := make([]error, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	m.Range(func(k KT, v VT) bool {
		wg.Add(1)
		go func() {
			if err := do(k, v); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			wg.Done()
		}()
		return true
	})
	wg.Wait()
	return errs
}

// RemoveAll removes all key-value pairs from the map where the value matches the given criteria.
//
// Parameters:
//
//	criteria: function to determine whether a value should be removed
//
// Returns:
//
//	nothing
func (m Map[KT, VT]) RemoveAll(criteria func(VT) bool) {
	m.Range(func(k KT, v VT) bool {
		if criteria(v) {
			m.Delete(k)
		}
		return true
	})
}

func (m Map[KT, VT]) Has(k KT) bool {
	_, ok := m.Load(k)
	return ok
}

// UnmarshalFromYAML unmarshals a yaml byte slice into the map.
//
// It overwrites all existing key-value pairs in the map.
//
// Parameters:
//
//	data: yaml byte slice to unmarshal
//
// Returns:
//
//	error: if the unmarshaling fails
func (m Map[KT, VT]) UnmarshalFromYAML(data []byte) error {
	if m.Size() != 0 {
		return errors.New("cannot unmarshal into non-empty map")
	}
	tmp := make(map[KT]VT)
	if err := yaml.Unmarshal(data, tmp); err != nil {
		return err
	}
	for k, v := range tmp {
		m.Store(k, v)
	}
	return nil
}

func (m Map[KT, VT]) String() string {
	tmp := make(map[KT]VT, m.Size())
	m.RangeAll(func(k KT, v VT) {
		tmp[k] = v
	})
	data, err := yaml.Marshal(tmp)
	if err != nil {
		return err.Error()
	}
	return string(data)
}
