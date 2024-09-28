package docker

import (
	"fmt"
	"testing"

	U "github.com/yusing/go-proxy/internal/utils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestNestedLabel(t *testing.T) {
	mName := "middleware1"
	mAttr := "prop1"
	v := "value1"
	pl, err := ParseLabel(makeLabel(NSProxy, "foo", fmt.Sprintf("%s.%s.%s", ProxyAttributeMiddlewares, mName, mAttr)), v)
	ExpectNoError(t, err.Error())
	sGot := ExpectType[*Label](t, pl.Value)
	ExpectFalse(t, sGot == nil)
	ExpectEqual(t, sGot.Namespace, mName)
	ExpectEqual(t, sGot.Attribute, mAttr)
}

func TestApplyNestedLabel(t *testing.T) {
	entry := new(struct {
		Middlewares NestedLabelMap `yaml:"middlewares"`
	})
	mName := "middleware1"
	mAttr := "prop1"
	v := "value1"
	pl, err := ParseLabel(makeLabel(NSProxy, "foo", fmt.Sprintf("%s.%s.%s", ProxyAttributeMiddlewares, mName, mAttr)), v)
	ExpectNoError(t, err.Error())
	err = ApplyLabel(entry, pl)
	ExpectNoError(t, err.Error())
	middleware1, ok := entry.Middlewares[mName]
	ExpectTrue(t, ok)
	got := ExpectType[string](t, middleware1[mAttr])
	ExpectEqual(t, got, v)
}

func TestApplyNestedLabelExisting(t *testing.T) {
	mName := "middleware1"
	mAttr := "prop1"
	v := "value1"

	checkAttr := "prop2"
	checkV := "value2"
	entry := new(struct {
		Middlewares NestedLabelMap `yaml:"middlewares"`
	})
	entry.Middlewares = make(NestedLabelMap)
	entry.Middlewares[mName] = make(U.SerializedObject)
	entry.Middlewares[mName][checkAttr] = checkV

	pl, err := ParseLabel(makeLabel(NSProxy, "foo", fmt.Sprintf("%s.%s.%s", ProxyAttributeMiddlewares, mName, mAttr)), v)
	ExpectNoError(t, err.Error())
	err = ApplyLabel(entry, pl)
	ExpectNoError(t, err.Error())
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
	mName := "middleware1"
	v := "value1"

	entry := new(struct {
		Middlewares NestedLabelMap `yaml:"middlewares"`
	})
	entry.Middlewares = make(NestedLabelMap)
	entry.Middlewares[mName] = make(U.SerializedObject)

	pl, err := ParseLabel(makeLabel(NSProxy, "foo", fmt.Sprintf("%s.%s", ProxyAttributeMiddlewares, mName)), v)
	ExpectNoError(t, err.Error())
	err = ApplyLabel(entry, pl)
	ExpectNoError(t, err.Error())
	_, ok := entry.Middlewares[mName]
	ExpectTrue(t, ok)
}
