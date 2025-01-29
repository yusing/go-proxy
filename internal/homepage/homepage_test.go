package homepage

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestOverrideItem(t *testing.T) {
	InitOverridesConfig()
	a := &Item{
		Alias: "foo",
		ItemConfig: &ItemConfig{
			Show: false,
			Name: "Foo",
			Icon: &IconURL{
				Value:      "/favicon.ico",
				IconSource: IconSourceRelative,
			},
			Category: "App",
		},
	}
	override := &ItemConfig{
		Show:     true,
		Name:     "Bar",
		Category: "Test",
		Icon: &IconURL{
			Value:      "@walkxcode/example.png",
			IconSource: IconSourceWalkXCode,
		},
	}
	overrides := GetOverrideConfig()
	overrides.OverrideItem(a.Alias, override)
	overridden := a.GetOverride()
	ExpectDeepEqual(t, overridden.ItemConfig, override)
}
