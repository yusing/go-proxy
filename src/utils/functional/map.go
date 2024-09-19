package functional

import (
	"github.com/puzpuzpuz/xsync/v3"
	"gopkg.in/yaml.v3"

	E "github.com/yusing/go-proxy/error"
)

type Map[KT comparable, VT any] struct {
	*xsync.MapOf[KT, VT]
}

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

// MergeFrom add contents from another `Map`, ignore duplicated keys
//
// Parameters:
// - other: `Map` of values to add from
//
// Return:
// - Map: a `Map` of duplicated keys-value pairs
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

func (m Map[KT, VT]) RangeAll(do func(k KT, v VT)) {
	m.Range(func(k KT, v VT) bool {
		do(k, v)
		return true
	})
}

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

func (m Map[KT, VT]) UnmarshalFromYAML(data []byte) E.NestedError {
	if m.Size() != 0 {
		return E.FailedWhy("unmarshal from yaml", "map is not empty")
	}
	tmp := make(map[KT]VT)
	if err := E.From(yaml.Unmarshal(data, tmp)); err.HasError() {
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
