package routevalidate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/route"
	rulepresets "github.com/yusing/godoxy/internal/route/rules/presets"
)

func TestValidateRulesRejectsResponsePhaseAuthentication(t *testing.T) {
	for _, action := range []string{"require_auth", `require_basic_auth "Admin"`} {
		for _, config := range []string{
			"status 200 { " + action + " }",
			"default {\n method POST {\n " + action + "\n set resp_header X-Test true\n }\n}",
		} {
			for _, source := range []string{"inline", "file"} {
				t.Run(action+"/"+config+"/"+source, func(t *testing.T) {
					r := &route.Route{Alias: "app"}
					if source == "inline" {
						require.NoError(t, r.Rules.Parse(config))
					} else {
						t.Chdir(t.TempDir())
						r.RuleFile = "rules.txt"
						require.NoError(t, os.WriteFile(r.RuleFile, []byte(config), 0600))
					}
					require.ErrorContains(t, validateRules(r), "request-only")
				})
			}
		}
	}
}

func TestValidateRulesPreservesRequestPhaseAuthentication(t *testing.T) {
	r := &route.Route{Alias: "app"}
	require.NoError(t, r.Rules.Parse("default { require_auth }"))
	require.NoError(t, validateRules(r))
}

func TestValidateRulesCopiesEmbeddedPreset(t *testing.T) {
	for _, name := range []string{"webui.yml", "webui_dev.yml"} {
		r := &route.Route{Alias: "app", RuleFile: "embed://" + name}
		require.NoError(t, validateRules(r))
		cached, ok := rulepresets.GetRulePreset(name)
		require.True(t, ok)
		require.NotEmpty(t, r.Rules)
		originalName := cached[0].Name
		r.Rules[0].Name = "modified"
		require.Equal(t, originalName, cached[0].Name)
	}
}
