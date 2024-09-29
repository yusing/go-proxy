package middleware

import (
	"net/http"

	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
)

type (
	modifyResponse struct {
		*modifyResponseOpts
		m *Middleware
	}
	// order: set_headers -> add_headers -> hide_headers
	modifyResponseOpts struct {
		SetHeaders  map[string]string
		AddHeaders  map[string]string
		HideHeaders []string
	}
)

var ModifyResponse = func() (mr *modifyResponse) {
	mr = new(modifyResponse)
	mr.m = new(Middleware)
	mr.m.labelParserMap = D.ValueParserMap{
		"set_headers":  D.YamlLikeMappingParser(true),
		"add_headers":  D.YamlLikeMappingParser(true),
		"hide_headers": D.YamlStringListParser,
	}
	mr.m.withOptions = NewModifyResponse
	return
}()

func NewModifyResponse(optsRaw OptionsRaw, _ *ReverseProxy) (*Middleware, E.NestedError) {
	mr := new(modifyResponse)
	mr.m = &Middleware{
		impl:           mr,
		modifyResponse: mr.modifyResponse,
	}
	mr.modifyResponseOpts = new(modifyResponseOpts)
	err := Deserialize(optsRaw, mr.modifyResponseOpts)
	if err != nil {
		return nil, E.FailWith("set options", err)
	}
	return mr.m, nil
}

func (mr *modifyResponse) modifyResponse(resp *http.Response) error {
	for k, v := range mr.SetHeaders {
		resp.Header.Set(k, v)
	}
	for k, v := range mr.AddHeaders {
		resp.Header.Add(k, v)
	}
	for _, k := range mr.HideHeaders {
		resp.Header.Del(k)
	}
	return nil
}
