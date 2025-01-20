package homepage

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestOverrideItem(t *testing.T) {
	// a := &Item{
	// 	Show:  false,
	// 	Alias: "foo",
	// 	Name:  "Foo",
	// 	Icon: &IconURL{
	// 		Value:      "/favicon.ico",
	// 		IconSource: IconSourceRelative,
	// 	},
	// 	Category: "App",
	// }
	// overrides := GetJSONConfig()
	// overrides.SetShowItemOverride(a.Alias, true)
	// overrides.SetDisplayNameOverride(a.Alias, "Bar")
	// overrides.SetDisplayCategoryOverride(a.Alias, "Test")
	// ExpectNoError(t, overrides.SetIconOverride(a.Alias, "@walkxcode/example.png"))

	// overridden := a.GetOverriddenItem()
	// ExpectTrue(t, overridden.Show)
	// ExpectEqual(t, overridden.Name, "Bar")
	// ExpectEqual(t, overridden.Category, "Test")
	// ExpectEqual(t, overridden.Icon.String(), "png/example.png")
}
