package functional_test

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/functional"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestNewMapFrom(t *testing.T) {
	m := NewMapFrom(map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	})
	ExpectEqual(t, m.Size(), 3)
	ExpectTrue(t, m.Has("a"))
	ExpectTrue(t, m.Has("b"))
	ExpectTrue(t, m.Has("c"))
}

func TestMapFind(t *testing.T) {
	m := NewMapFrom(map[string]map[string]int{
		"a": {
			"a": 1,
		},
		"b": {
			"a": 1,
			"b": 2,
		},
		"c": {
			"b": 2,
			"c": 3,
		},
	})
	res := MapFind(m, func(inner map[string]int) (int, bool) {
		if _, ok := inner["c"]; ok && inner["c"] == 3 {
			return inner["c"], true
		}
		return 0, false
	})
	ExpectEqual(t, res, 3)
}

func TestMergeFrom(t *testing.T) {
	m1 := NewMapFrom(map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
		"d": 4,
	})
	m2 := NewMapFrom(map[string]int{
		"a": 1,
		"c": 123,
		"e": 456,
		"f": 6,
	})
	dup := m1.MergeFrom(m2)

	ExpectEqual(t, m1.Size(), 6)
	ExpectTrue(t, m1.Has("e"))
	ExpectTrue(t, m1.Has("f"))
	c, _ := m1.Load("c")
	d, _ := m1.Load("d")
	e, _ := m1.Load("e")
	f, _ := m1.Load("f")
	ExpectEqual(t, c, 3)
	ExpectEqual(t, d, 4)
	ExpectEqual(t, e, 456)
	ExpectEqual(t, f, 6)

	ExpectEqual(t, dup.Size(), 2)
	ExpectTrue(t, dup.Has("a"))
	ExpectTrue(t, dup.Has("c"))
}
