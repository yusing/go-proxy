package error

import "testing"

func TestBuilder(t *testing.T) {
	eb := NewBuilder("error occurred")
	eb.Add(Failure("Action 1").With(Invalid("Inner", "1")).With(Invalid("Inner", "2")))
	eb.Add(Failure("Action 2").With(Invalid("Inner", "3")))

	got := eb.Build().Error()
	expected1 :=
		(`error occurred:
  - Action 1 failed:
    - invalid Inner - 1
    - invalid Inner - 2
  - Action 2 failed:
    - invalid Inner - 3`)
	expected2 :=
		(`error occurred:
  - Action 1 failed:
    - invalid Inner - 2
    - invalid Inner - 1
  - Action 2 failed:
    - invalid Inner - 3`)
	if got != expected1 && got != expected2 {
		t.Errorf("expected \n%s, got \n%s", expected1, got)
	}
}
