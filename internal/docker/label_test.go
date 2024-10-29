package docker

import (
	"fmt"
	"testing"

	U "github.com/yusing/go-proxy/internal/utils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

const (
	mName = "middleware1"
	mAttr = "prop1"
	v     = "value1"
)

func makeLabel(ns, name, attr string) string {
	return fmt.Sprintf("%s.%s.%s", ns, name, attr)
}

func TestNestedLabel(t *testing.T) {
	mAttr := "prop1"
	lbl := ParseLabel(makeLabel(NSProxy, "foo", makeLabel("middlewares", mName, mAttr)), v)
	sGot := ExpectType[*Label](t, lbl.Value)
	ExpectFalse(t, sGot == nil)
	ExpectEqual(t, sGot.Namespace, mName)
	ExpectEqual(t, sGot.Attribute, mAttr)
}

func TestApplyNestedLabel(t *testing.T) {
	entry := new(struct {
		Middlewares NestedLabelMap `yaml:"middlewares"`
	})
	lbl := ParseLabel(makeLabel(NSProxy, "foo", makeLabel("middlewares", mName, mAttr)), v)
	err := ApplyLabel(entry, lbl)
	ExpectNoError(t, err)
	middleware1, ok := entry.Middlewares[mName]
	ExpectTrue(t, ok)
	got := ExpectType[string](t, middleware1[mAttr])
	ExpectEqual(t, got, v)
}

func TestApplyNestedLabelExisting(t *testing.T) {
	checkAttr := "prop2"
	checkV := "value2"
	entry := new(struct {
		Middlewares NestedLabelMap `yaml:"middlewares"`
	})
	entry.Middlewares = make(NestedLabelMap)
	entry.Middlewares[mName] = make(U.SerializedObject)
	entry.Middlewares[mName][checkAttr] = checkV

	lbl := ParseLabel(makeLabel(NSProxy, "foo", makeLabel("middlewares", mName, mAttr)), v)
	err := ApplyLabel(entry, lbl)
	ExpectNoError(t, err)
	middleware1, ok := entry.Middlewares[mName]
	ExpectTrue(t, ok)
	got := ExpectType[string](t, middleware1[mAttr])
	ExpectEqual(t, got, v)

	// check if prop2 is affected
	ExpectFalse(t, middleware1[checkAttr] == nil)
	got = ExpectType[string](t, middleware1[checkAttr])
	ExpectEqual(t, got, checkV)
}

func TestApplyNestedLabelNoAttr(t *testing.T) {
	entry := new(struct {
		Middlewares NestedLabelMap `yaml:"middlewares"`
	})
	entry.Middlewares = make(NestedLabelMap)
	entry.Middlewares[mName] = make(U.SerializedObject)

	lbl := ParseLabel(makeLabel(NSProxy, "foo", fmt.Sprintf("%s.%s", "middlewares", mName)), v)
	err := ApplyLabel(entry, lbl)
	ExpectNoError(t, err)
	_, ok := entry.Middlewares[mName]
	ExpectTrue(t, ok)
}
