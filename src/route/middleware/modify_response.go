package middleware

import (
	"net/http"

	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	U "github.com/yusing/go-proxy/utils"
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

var ModifyResponse = newModifyResponse()

func newModifyResponse() (mr *modifyResponse) {
	mr = new(modifyResponse)
	mr.m = new(Middleware)
	mr.m.labelParserMap = D.ValueParserMap{
		"set_headers":  D.YamlLikeMappingParser(true),
		"add_headers":  D.YamlLikeMappingParser(true),
		"hide_headers": D.YamlStringListParser,
	}
	mr.m.withOptions = func(optsRaw OptionsRaw, rp *ReverseProxy) (*Middleware, E.NestedError) {
		mrWithOpts := new(modifyResponse)
		mrWithOpts.m = &Middleware{
			impl:           mrWithOpts,
			modifyResponse: mrWithOpts.modifyResponse,
		}
		mrWithOpts.modifyResponseOpts = new(modifyResponseOpts)
		err := U.Deserialize(optsRaw, mrWithOpts.modifyResponseOpts)
		if err != nil {
			return nil, E.FailWith("set options", err)
		}
		return mrWithOpts.m, nil
	}
	return
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