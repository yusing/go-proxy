package v1

import (
	"net/http"
	"strconv"

	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/utils"
)

const (
	HomepageOverrideItem          = "item"
	HomepageOverrideCategoryOrder = "category_order"
	HomepageOverrideCategoryName  = "category_name"
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
