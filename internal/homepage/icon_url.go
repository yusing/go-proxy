package homepage

import (
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type IconURL struct {
	Value      string `json:"value"`
	IsRelative bool   `json:"is_relative"`
}

const DashboardIconBaseURL = "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/"

var ErrInvalidIconURL = E.New("invalid icon url")

// Parse implements strutils.Parser.
func (u *IconURL) Parse(v string) error {
	if v == "" {
		return ErrInvalidIconURL
	}
	slashIndex := strings.Index(v, "/")
	if slashIndex == -1 {
		return ErrInvalidIconURL
	}
	beforeSlash := v[:slashIndex]
	switch beforeSlash {
	case "http:", "https:":
		u.Value = v
		return nil
	case "@target":
		u.Value = v[slashIndex:]
		u.IsRelative = true
		return nil
	case "png", "svg": // walkXCode Icons
		u.Value = DashboardIconBaseURL + v
		return nil
	default:
		return ErrInvalidIconURL
	}
}

func (u *IconURL) String() string {
	return u.Value
}
