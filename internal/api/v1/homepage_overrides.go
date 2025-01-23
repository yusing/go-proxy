package v1

import (
	"net/http"
	"strconv"

	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

const (
	HomepageOverrideItem          = "item"
	HomepageOverrideCategoryOrder = "category_order"
	HomepageOverrideCategoryName  = "category_name"
	HomepageOverrideItemVisible   = "item_visible"
)

func SetHomePageOverrides(w http.ResponseWriter, r *http.Request) {
	what, which, value := r.FormValue("what"), r.FormValue("which"), r.FormValue("value")
	if what == "" || which == "" {
		http.Error(w, "missing what or which", http.StatusBadRequest)
		return
	}
	if value == "" {
		http.Error(w, "missing value", http.StatusBadRequest)
		return
	}
	overrides := homepage.GetOverrideConfig()
	switch what {
	case HomepageOverrideItem:
		var override homepage.ItemConfig
		if err := utils.DeserializeJSON([]byte(value), &override); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		overrides.OverrideItem(which, &override)
	case HomepageOverrideItemVisible: // POST /v1/item_visible [a,b,c], false => hide a, b, c
		keys := strutils.CommaSeperatedList(which)
		if strutils.ParseBool(value) {
			overrides.UnhideItems(keys...)
		} else {
			overrides.HideItems(keys...)
		}
	case HomepageOverrideCategoryName:
		overrides.SetCategoryNameOverride(which, value)
	case HomepageOverrideCategoryOrder:
		v, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, "invalid integer", http.StatusBadRequest)
			return
		}
		overrides.SetCategoryOrder(which, v)
	default:
		http.Error(w, "invalid what", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
