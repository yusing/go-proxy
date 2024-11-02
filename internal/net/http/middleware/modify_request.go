package middleware

import (
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
)

type (
	modifyRequest struct {
		modifyRequestOpts
		m *Middleware
	}
	// order: set_headers -> add_headers -> hide_headers
	modifyRequestOpts struct {
		SetHeaders  map[string]string `json:"setHeaders"`
		AddHeaders  map[string]string `json:"addHeaders"`
		HideHeaders []string          `json:"hideHeaders"`
	}
)

var ModifyRequest = &Middleware{withOptions: NewModifyRequest}

func NewModifyRequest(optsRaw OptionsRaw) (*Middleware, E.Error) {
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
	err := Deserialize(optsRaw, &mr.modifyRequestOpts)
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
