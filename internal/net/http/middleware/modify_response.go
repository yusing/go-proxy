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
		modifyResponse: func(resp *http.Response) error {
			mr.m.AddTraceResponse("before modify response", resp)
			mr.modifyHeaders(resp.Request, resp, resp.Header)
			mr.m.AddTraceResponse("after modify response", resp)
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
