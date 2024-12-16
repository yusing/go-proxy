package utils

import (
	"errors"
	"reflect"
	"testing"

	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestSerializeDeserialize(t *testing.T) {
	type S struct {
		I   int
		S   string
		IS  []int
		SS  []string
		MSI map[string]int
		MIS map[int]string
	}

	var (
		testStruct = S{
			I:   1,
			S:   "hello",
			IS:  []int{1, 2, 3},
			SS:  []string{"a", "b", "c"},
			MSI: map[string]int{"a": 1, "b": 2, "c": 3},
			MIS: map[int]string{1: "a", 2: "b", 3: "c"},
		}
		testStructSerialized = map[string]any{
			"I":   1,
			"S":   "hello",
			"IS":  []int{1, 2, 3},
			"SS":  []string{"a", "b", "c"},
			"MSI": map[string]int{"a": 1, "b": 2, "c": 3},
			"MIS": map[int]string{1: "a", 2: "b", 3: "c"},
		}
	)

	t.Run("serialize", func(t *testing.T) {
		s, err := Serialize(testStruct)
		ExpectNoError(t, err)
		ExpectDeepEqual(t, s, testStructSerialized)
	})

	t.Run("deserialize", func(t *testing.T) {
		var s2 S
		err := Deserialize(testStructSerialized, &s2)
		ExpectNoError(t, err)
		ExpectDeepEqual(t, s2, testStruct)
	})
}

func TestDeserializeAnonymousField(t *testing.T) {
	type Anon struct {
		A, B int
	}
	var s struct {
		Anon
		C int
	}
	err := Deserialize(map[string]any{"a": 1, "b": 2, "c": 3}, &s)
	ExpectNoError(t, err)
	ExpectEqual(t, s.A, 1)
	ExpectEqual(t, s.B, 2)
	ExpectEqual(t, s.C, 3)
}

func TestStringIntConvert(t *testing.T) {
	s := "127"

	test := struct {
		i8  int8
		i16 int16
		i32 int32
		i64 int64
		u8  uint8
		u16 uint16
		u32 uint32
		u64 uint64
	}{}

	ok, err := ConvertString(s, reflect.ValueOf(&test.i8))

	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.i8, int8(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.i16))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.i16, int16(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.i32))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.i32, int32(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.i64))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.i64, int64(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u8))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.u8, uint8(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u16))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.u16, uint16(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u32))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.u32, uint32(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u64))
	ExpectTrue(t, ok)
	ExpectNoError(t, err)
	ExpectEqual(t, test.u64, uint64(127))
}

type testModel struct {
	Test testType
}

type testType struct {
	foo int
	bar string
}

var errInvalid = errors.New("invalid input type")

func (c *testType) ConvertFrom(v any) E.Error {
	switch v := v.(type) {
	case string:
		c.bar = v
		return nil
	case int:
		c.foo = v
		return nil
	default:
		return E.Errorf("%w %T", errInvalid, v)
	}
}

func TestConvertor(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		m := new(testModel)
		ExpectNoError(t, Deserialize(map[string]any{"Test": "bar"}, m))

		ExpectEqual(t, m.Test.foo, 0)
		ExpectEqual(t, m.Test.bar, "bar")
	})

	t.Run("int", func(t *testing.T) {
		m := new(testModel)
		ExpectNoError(t, Deserialize(map[string]any{"Test": 123}, m))

		ExpectEqual(t, m.Test.foo, 123)
		ExpectEqual(t, m.Test.bar, "")
	})

	t.Run("invalid", func(t *testing.T) {
		m := new(testModel)
		ExpectError(t, errInvalid, Deserialize(map[string]any{"Test": 123.456}, m))
	})
}
