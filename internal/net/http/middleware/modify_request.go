package middleware

import (
	"net/http"
	"strings"
)

type (
	modifyRequest struct {
		ModifyRequestOpts
		*Tracer
	}
	// order: set_headers -> add_headers -> hide_headers
	ModifyRequestOpts struct {
		SetHeaders  map[string]string
		AddHeaders  map[string]string
		HideHeaders []string

		needVarSubstitution bool
	}
)

var ModifyRequest = NewMiddleware[modifyRequest]()

// finalize implements MiddlewareFinalizer.
func (mr *ModifyRequestOpts) finalize() {
	mr.checkVarSubstitution()
}

// before implements RequestModifier.
func (mr *modifyRequest) before(w http.ResponseWriter, r *http.Request) (proceed bool) {
	mr.AddTraceRequest("before modify request", r)
	mr.modifyHeaders(r, nil, r.Header)
	mr.AddTraceRequest("after modify request", r)
	return true
}

func (mr *ModifyRequestOpts) checkVarSubstitution() {
	for _, m := range []map[string]string{mr.SetHeaders, mr.AddHeaders} {
		for _, v := range m {
			if strings.ContainsRune(v, '$') {
				mr.needVarSubstitution = true
				return
			}
		}
	}
}

func (mr *ModifyRequestOpts) modifyHeaders(req *http.Request, resp *http.Response, headers http.Header) {
	if !mr.needVarSubstitution {
		for k, v := range mr.SetHeaders {
			if req != nil && strings.EqualFold(k, "host") {
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
			if req != nil && strings.EqualFold(k, "host") {
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
