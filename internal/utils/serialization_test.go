package utils

import (
	"reflect"
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

type S = struct {
	I   int
	S   string
	IS  []int
	SS  []string
	MSI map[string]int
	MIS map[int]string
}

var testStruct = S{
	I:   1,
	S:   "hello",
	IS:  []int{1, 2, 3},
	SS:  []string{"a", "b", "c"},
	MSI: map[string]int{"a": 1, "b": 2, "c": 3},
	MIS: map[int]string{1: "a", 2: "b", 3: "c"},
}

var testStructSerialized = map[string]any{
	"I":   1,
	"S":   "hello",
	"IS":  []int{1, 2, 3},
	"SS":  []string{"a", "b", "c"},
	"MSI": map[string]int{"a": 1, "b": 2, "c": 3},
	"MIS": map[int]string{1: "a", 2: "b", 3: "c"},
}

func TestSerialize(t *testing.T) {
	s, err := Serialize(testStruct)
	ExpectNoError(t, err.Error())
	ExpectDeepEqual(t, s, testStructSerialized)
}

func TestDeserialize(t *testing.T) {
	var s S
	err := Deserialize(testStructSerialized, &s)
	ExpectNoError(t, err.Error())
	ExpectDeepEqual(t, s, testStruct)
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
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.i8, int8(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.i16))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.i16, int16(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.i32))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.i32, int32(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.i64))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.i64, int64(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u8))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.u8, uint8(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u16))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.u16, uint16(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u32))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.u32, uint32(127))

	ok, err = ConvertString(s, reflect.ValueOf(&test.u64))
	ExpectTrue(t, ok)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, test.u64, uint64(127))
}
