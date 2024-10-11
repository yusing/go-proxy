package error_test

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestBuilderEmpty(t *testing.T) {
	eb := NewBuilder("qwer")
	ExpectTrue(t, eb.Build() == nil)
	ExpectTrue(t, eb.Build().NoError())
	ExpectFalse(t, eb.HasError())
}

func TestBuilderAddNil(t *testing.T) {
	eb := NewBuilder("asdf")
	var err NestedError
	for range 3 {
		eb.Add(nil)
	}
	for range 3 {
		eb.Add(err)
	}
	ExpectTrue(t, eb.Build() == nil)
	ExpectTrue(t, eb.Build().NoError())
	ExpectFalse(t, eb.HasError())
}

func TestBuilderNested(t *testing.T) {
	eb := NewBuilder("error occurred")
	eb.Add(Failure("Action 1").With(Invalid("Inner", "1")).With(Invalid("Inner", "2")))
	eb.Add(Failure("Action 2").With(Invalid("Inner", "3")))

	got := eb.Build().String()
	expected1 := (`error occurred:
  - Action 1 failed:
    - invalid Inner: 1
    - invalid Inner: 2
  - Action 2 failed:
    - invalid Inner: 3`)
	expected2 := (`error occurred:
  - Action 1 failed:
    - invalid Inner: "1"
    - invalid Inner: "2"
  - Action 2 failed:
    - invalid Inner: "3"`)
	if got != expected1 && got != expected2 {
		t.Errorf("expected \n%s, got \n%s", expected1, got)
	}
}
