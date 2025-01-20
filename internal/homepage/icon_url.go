package homepage

import (
	"strings"

	"github.com/yusing/go-proxy/internal"
	E "github.com/yusing/go-proxy/internal/error"
)

type (
	IconURL struct {
		Value      string `json:"value"`
		IconSource `json:"source"`
		Extra      *IconExtra `json:"extra"`
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
	IconSourceSelfhSt
)

var ErrInvalidIconURL = E.New("invalid icon url")

func (u *IconURL) HasIcon() bool {
	if u.IconSource == IconSourceSelfhSt &&
		!internal.HasSelfhstIcon(u.Extra.Name, u.Extra.FileType) {
		return false
	}
	if u.IconSource == IconSourceWalkXCode &&
		!internal.HasWalkxCodeIcon(u.Extra.Name, u.Extra.FileType) {
		return false
	}
	return true
}

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
	case "@target", "": // @target/favicon.ico, /favicon.ico
		u.Value = v[slashIndex:]
		u.IconSource = IconSourceRelative
		if u.Value == "/" {
			return ErrInvalidIconURL.Withf("%s", "empty path")
		}
	case "png", "svg", "webp": // walkXCode Icons
		u.Value = v
		u.IconSource = IconSourceWalkXCode
		u.Extra = &IconExtra{
			FileType: beforeSlash,
			Name:     strings.TrimSuffix(v[slashIndex+1:], "."+beforeSlash),
		}
	case "@selfhst": // selfh.st Icons, @selfhst/<reference>.<format>
		u.Value = v[slashIndex:]
		u.IconSource = IconSourceSelfhSt
		parts := strings.Split(v[slashIndex+1:], ".")
		if len(parts) != 2 {
			return ErrInvalidIconURL.Withf("%s", "expect @selfhst/<reference>.<format>, e.g. @selfhst/adguard-home.webp")
		}
		reference, format := parts[0], strings.ToLower(parts[1])
		if reference == "" || format == "" {
			return ErrInvalidIconURL
		}
		switch format {
		case "svg", "png", "webp":
		default:
			return ErrInvalidIconURL.Withf("%s", "invalid format, expect svg/png/webp")
		}
		u.Extra = &IconExtra{
			FileType: format,
			Name:     reference,
		}
	default:
		return ErrInvalidIconURL.Withf("%s", v)
	}

	if u.Value == "" {
		return ErrInvalidIconURL.Withf("%s", "empty")
	}

	if !u.HasIcon() {
		return ErrInvalidIconURL.Withf("no such icon %s", u.Value)
	}
	return nil
}

func (u *IconURL) String() string {
	return u.Value
}
