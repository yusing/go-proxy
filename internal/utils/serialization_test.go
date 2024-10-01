package utils

import (
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
