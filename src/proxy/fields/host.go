package fields

import (
	E "github.com/yusing/go-proxy/error"
	F "github.com/yusing/go-proxy/utils/functional"
)

type Host struct{ F.Stringable }
type Subdomain = Alias

func NewHost(s string) (Host, E.NestedError) {
	return Host{F.NewStringable(s)}, E.Nil()
}

func (h Host) Subdomain() (*Subdomain, E.NestedError) {
	if i := h.IndexRune(':'); i != -1 {
		return &Subdomain{h.SubStr(0, i)}, E.Nil()
	}
	return nil, E.Invalid("host", h)
}
