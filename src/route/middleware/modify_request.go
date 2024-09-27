package middleware

import (
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	U "github.com/yusing/go-proxy/utils"
)

type (
	modifyRequest struct {
		*modifyRequestOpts
		m *Middleware
	}
	// order: set_headers -> add_headers -> hide_headers
	modifyRequestOpts struct {
		SetHeaders  map[string]string
		AddHeaders  map[string]string
		HideHeaders []string
	}
)

var ModifyRequest = newModifyRequest()

func newModifyRequest() (mr *modifyRequest) {
	mr = new(modifyRequest)
	mr.m = new(Middleware)
	mr.m.labelParserMap = D.ValueParserMap{
		"set_headers":  D.YamlLikeMappingParser(true),
		"add_headers":  D.YamlLikeMappingParser(true),
		"hide_headers": D.YamlStringListParser,
	}
	mr.m.withOptions = func(optsRaw OptionsRaw, rp *ReverseProxy) (*Middleware, E.NestedError) {
		mrWithOpts := new(modifyRequest)
		mrWithOpts.m = &Middleware{
			impl:    mrWithOpts,
			rewrite: mrWithOpts.modifyRequest,
		}
		mrWithOpts.modifyRequestOpts = new(modifyRequestOpts)
		err := U.Deserialize(optsRaw, mrWithOpts.modifyRequestOpts)
		if err != nil {
			return nil, E.FailWith("set options", err)
		}
		return mrWithOpts.m, nil
	}
	return
}

func (mr *modifyRequest) modifyRequest(req *ProxyRequest) {
	for k, v := range mr.SetHeaders {
		req.Out.Header.Set(k, v)
	}
	for k, v := range mr.AddHeaders {
		req.Out.Header.Add(k, v)
	}
	for _, k := range mr.HideHeaders {
		req.Out.Header.Del(k)
	}
}
