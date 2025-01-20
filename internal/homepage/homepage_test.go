package homepage

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestOverrideItem(t *testing.T) {
	a := &Item{
		Show:  false,
		Alias: "foo",
		Name:  "Foo",
		Icon: &IconURL{
			Value:      "/favicon.ico",
			IconSource: IconSourceRelative,
		},
		Category: "App",
	}
	overrides := GetJSONConfig()
	ExpectNoError(t, overrides.SetShowItemOverride(a.Alias, true))
	ExpectNoError(t, overrides.SetDisplayNameOverride(a.Alias, "Bar"))
	ExpectNoError(t, overrides.SetDisplayCategoryOverride(a.Alias, "Test"))
	ExpectNoError(t, overrides.SetIconOverride(a.Alias, "png/example.png"))

	overridden := a.GetOverriddenItem()
	ExpectTrue(t, overridden.Show)
	ExpectEqual(t, overridden.Name, "Bar")
	ExpectEqual(t, overridden.Category, "Test")
	ExpectEqual(t, overridden.Icon.String(), "png/example.png")
}
