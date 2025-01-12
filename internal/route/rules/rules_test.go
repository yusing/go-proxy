package rules

import (
	"testing"

	"github.com/yusing/go-proxy/internal/utils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestParseRule(t *testing.T) {
	test := []map[string]any{
		{
			"name": "test",
			"on":   "method POST",
			"do":   "error 403 Forbidden",
		},
		{
			"name": "auth",
			"on":   `basic_auth "username" "password" | basic_auth username2 "password2" | basic_auth "username3" "password3"`,
			"do":   "bypass",
		},
		{
			"name": "default",
			"do":   "require_basic_auth any_realm",
		},
	}

	var rules struct {
		Rules Rules
	}
	err := utils.Deserialize(utils.SerializedObject{"rules": test}, &rules)
	ExpectNoError(t, err)
	ExpectEqual(t, len(rules.Rules), len(test))
	ExpectEqual(t, rules.Rules[0].Name, "test")
	ExpectEqual(t, rules.Rules[0].On.String(), "method POST")
	ExpectEqual(t, rules.Rules[0].Do.String(), "error 403 Forbidden")

	ExpectEqual(t, rules.Rules[1].Name, "auth")
	ExpectEqual(t, rules.Rules[1].On.String(), `basic_auth "username" "password" | basic_auth username2 "password2" | basic_auth "username3" "password3"`)
	ExpectEqual(t, rules.Rules[1].Do.String(), "bypass")

	ExpectEqual(t, rules.Rules[2].Name, "default")
	ExpectEqual(t, rules.Rules[2].Do.String(), "require_basic_auth any_realm")
}

// TODO: real tests.
