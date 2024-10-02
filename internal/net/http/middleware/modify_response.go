package middleware

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
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

var ModifyResponse = &modifyResponse{
	m: &Middleware{withOptions: NewModifyResponse},
}

func NewModifyResponse(optsRaw OptionsRaw) (*Middleware, E.NestedError) {
	mr := new(modifyResponse)
	mr.m = &Middleware{impl: mr}
	if common.IsDebug {
		mr.m.modifyResponse = mr.modifyResponseWithTrace
	} else {
		mr.m.modifyResponse = mr.modifyResponse
	}
	mr.modifyResponseOpts = new(modifyResponseOpts)
	err := Deserialize(optsRaw, mr.modifyResponseOpts)
	if err != nil {
		return nil, err
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

func (mr *modifyResponse) modifyResponseWithTrace(resp *http.Response) error {
	mr.m.AddTraceResponse("before modify response", resp)
	err := mr.modifyResponse(resp)
	mr.m.AddTraceResponse("after modify response", resp)
	return err
}
