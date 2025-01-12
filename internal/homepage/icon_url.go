package homepage

import (
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	IconURL struct {
		Value string `json:"value"`
		IconSource
		Extra *IconExtra `json:"extra"`
	}

	IconExtra struct {
		FileType string `json:"file_type"`
		Name     string `json:"name"`
	}

	IconSource int
)

const (
	IconSourceAbsolute IconSource = iota
	IconSourceRelative
	IconSourceWalkXCode
)

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
		u.IconSource = IconSourceAbsolute
		return nil
	case "@target":
		u.Value = v[slashIndex:]
		u.IconSource = IconSourceRelative
		return nil
	case "png", "svg", "webp": // walkXCode Icons
		u.Value = v
		u.IconSource = IconSourceWalkXCode
		u.Extra = &IconExtra{
			FileType: beforeSlash,
			Name:     strings.TrimSuffix(v[slashIndex+1:], "."+beforeSlash),
		}
		return nil
	default:
		return ErrInvalidIconURL
	}
}

func (u *IconURL) String() string {
	return u.Value
}
