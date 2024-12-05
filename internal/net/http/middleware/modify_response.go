package middleware

import (
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
)

type modifyResponse = modifyRequest

var ModifyResponse = &Middleware{withOptions: NewModifyResponse}

func NewModifyResponse(optsRaw OptionsRaw) (*Middleware, E.Error) {
	mr := new(modifyResponse)
	mr.m = &Middleware{
		impl: mr,
		before: func(next http.HandlerFunc, w ResponseWriter, r *Request) {
			next(w, r)
		},
		modifyResponse: func(resp *Response) error {
			mr.m.AddTraceResponse("before modify response", resp.Response)
			mr.modifyHeaders(resp.OriginalRequest, resp, resp.Header)
			mr.m.AddTraceResponse("after modify response", resp.Response)
			return nil
		},
	}
	err := Deserialize(optsRaw, &mr.modifyRequestOpts)
	if err != nil {
		return nil, err
	}
	mr.checkVarSubstitution()
	return mr.m, nil
}
