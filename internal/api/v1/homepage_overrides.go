package v1

import (
	"net/http"
	"strconv"

	"github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

const (
	HomepageOverrideDisplayname     = "display_name"
	HomepageOverrideDisplayOrder    = "display_order"
	HomepageOverrideDisplayCategory = "display_category"
	HomepageOverrideCategoryOrder   = "category_order"
	HomepageOverrideCategoryName    = "category_name"
	HomepageOverrideIcon            = "icon"
	HomepageOverrideShow            = "show"
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
	overrides := homepage.GetJSONConfig()
	switch what {
	case HomepageOverrideDisplayname:
		utils.RespondError(w, overrides.SetDisplayNameOverride(which, value))
	case HomepageOverrideDisplayCategory:
		utils.RespondError(w, overrides.SetDisplayCategoryOverride(which, value))
	case HomepageOverrideCategoryName:
		utils.RespondError(w, overrides.SetCategoryNameOverride(which, value))
	case HomepageOverrideIcon:
		utils.RespondError(w, overrides.SetIconOverride(which, value))
	case HomepageOverrideShow:
		utils.RespondError(w, overrides.SetShowItemOverride(which, strutils.ParseBool(value)))
	case HomepageOverrideDisplayOrder, HomepageOverrideCategoryOrder:
		v, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, "invalid integer", http.StatusBadRequest)
			return
		}
		if what == HomepageOverrideDisplayOrder {
			utils.RespondError(w, overrides.SetDisplayOrder(which, v))
		} else {
			utils.RespondError(w, overrides.SetCategoryOrder(which, v))
		}
	default:
		http.Error(w, "invalid what", http.StatusBadRequest)
	}
}
