package middleware

import (
	"net/http"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	modifyRequest struct {
		modifyRequestOpts
		m                   *Middleware
		needVarSubstitution bool
	}
	// order: set_headers -> add_headers -> hide_headers
	modifyRequestOpts struct {
		SetHeaders  map[string]string
		AddHeaders  map[string]string
		HideHeaders []string
	}
)

var ModifyRequest = &Middleware{withOptions: NewModifyRequest}

func NewModifyRequest(optsRaw OptionsRaw) (*Middleware, E.Error) {
	mr := new(modifyRequest)
	mr.m = &Middleware{
		impl: mr,
		before: Rewrite(func(req *Request) {
			mr.m.AddTraceRequest("before modify request", req)
			mr.modifyHeaders(req, nil, req.Header)
			mr.m.AddTraceRequest("after modify request", req)
		}),
	}
	err := Deserialize(optsRaw, &mr.modifyRequestOpts)
	if err != nil {
		return nil, err
	}
	mr.checkVarSubstitution()
	return mr.m, nil
}

func (mr *modifyRequest) checkVarSubstitution() {
	for _, m := range []map[string]string{mr.SetHeaders, mr.AddHeaders} {
		for _, v := range m {
			if strings.ContainsRune(v, '$') {
				mr.needVarSubstitution = true
				return
			}
		}
	}
}

func (mr *modifyRequest) modifyHeaders(req *Request, resp *Response, headers http.Header) {
	if !mr.needVarSubstitution {
		for k, v := range mr.SetHeaders {
			if req != nil && strings.ToLower(k) == "host" {
				defer func() {
					req.Host = v
				}()
			}
			headers.Set(k, v)
		}
		for k, v := range mr.AddHeaders {
			headers.Add(k, v)
		}
	} else {
		for k, v := range mr.SetHeaders {
			if req != nil && strings.ToLower(k) == "host" {
				defer func() {
					req.Host = varReplace(req, resp, v)
				}()
			}
			headers.Set(k, varReplace(req, resp, v))
		}
		for k, v := range mr.AddHeaders {
			headers.Add(k, varReplace(req, resp, v))
		}
	}

	for _, k := range mr.HideHeaders {
		headers.Del(k)
	}
}
