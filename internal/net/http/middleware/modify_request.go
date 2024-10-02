package middleware

import (
	"github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
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

var ModifyRequest = func() *modifyRequest {
	mr := new(modifyRequest)
	mr.m = new(Middleware)
	mr.m.labelParserMap = D.ValueParserMap{
		"set_headers":  D.YamlLikeMappingParser(true),
		"add_headers":  D.YamlLikeMappingParser(true),
		"hide_headers": D.YamlStringListParser,
	}
	mr.m.withOptions = NewModifyRequest
	return mr
}()

func NewModifyRequest(optsRaw OptionsRaw) (*Middleware, E.NestedError) {
	mr := new(modifyRequest)
	var mrFunc RewriteFunc
	if common.IsDebug {
		mrFunc = mr.modifyRequestWithTrace
	} else {
		mrFunc = mr.modifyRequest
	}
	mr.m = &Middleware{
		impl:   mr,
		before: Rewrite(mrFunc),
	}
	mr.modifyRequestOpts = new(modifyRequestOpts)
	err := Deserialize(optsRaw, mr.modifyRequestOpts)
	if err != nil {
		return nil, err
	}
	return mr.m, nil
}

func (mr *modifyRequest) modifyRequest(req *Request) {
	for k, v := range mr.SetHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range mr.AddHeaders {
		req.Header.Add(k, v)
	}
	for _, k := range mr.HideHeaders {
		req.Header.Del(k)
	}
}

func (mr *modifyRequest) modifyRequestWithTrace(req *Request) {
	mr.m.AddTraceRequest("before modify request", req)
	mr.modifyRequest(req)
	mr.m.AddTraceRequest("after modify request", req)
}
