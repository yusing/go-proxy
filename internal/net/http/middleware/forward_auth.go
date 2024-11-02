// Modified from Traefik Labs's MIT-licensed code (https://github.com/traefik/traefik/blob/master/pkg/middlewares/auth/forward.go)
// Copyright (c) 2020-2024 Traefik Labs
// Copyright (c) 2024 yusing

package middleware

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

type (
	forwardAuth struct {
		forwardAuthOpts
		m      *Middleware
		client http.Client
	}
	forwardAuthOpts struct {
		Address                  string   `json:"address"`
		TrustForwardHeader       bool     `json:"trustForwardHeader"`
		AuthResponseHeaders      []string `json:"authResponseHeaders"`
		AddAuthCookiesToResponse []string `json:"addAuthCookiesToResponse"`

		transport http.RoundTripper
	}
)

var ForwardAuth = &Middleware{withOptions: NewForwardAuthfunc}

func NewForwardAuthfunc(optsRaw OptionsRaw) (*Middleware, E.Error) {
	fa := new(forwardAuth)
	if err := Deserialize(optsRaw, &fa.forwardAuthOpts); err != nil {
		return nil, err
	}
	if _, err := url.Parse(fa.Address); err != nil {
		return nil, E.From(err)
	}

	fa.m = &Middleware{
		impl:   fa,
		before: fa.forward,
	}

	// TODO: use tr from reverse proxy
	tr, ok := fa.transport.(*http.Transport)
	if ok {
		tr = tr.Clone()
	} else {
		tr = gphttp.DefaultTransport.Clone()
	}

	fa.client = http.Client{
		CheckRedirect: func(r *Request, via []*Request) error {
			return http.ErrUseLastResponse
		},
		Timeout:   30 * time.Second,
		Transport: tr,
	}
	return fa.m, nil
}

func (fa *forwardAuth) forward(next http.HandlerFunc, w ResponseWriter, req *Request) {
	gphttp.RemoveHop(req.Header)

	faReq, err := http.NewRequestWithContext(
		req.Context(),
		http.MethodGet,
		fa.Address,
		nil,
	)
	if err != nil {
		fa.m.AddTracef("new request err to %s", fa.Address).WithError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gphttp.CopyHeader(faReq.Header, req.Header)
	gphttp.RemoveHop(faReq.Header)

	faReq.Header = gphttp.FilterHeaders(faReq.Header, fa.AuthResponseHeaders)
	fa.setAuthHeaders(req, faReq)
	fa.m.AddTraceRequest("forward auth request", faReq)

	faResp, err := fa.client.Do(faReq)
	if err != nil {
		fa.m.AddTracef("failed to call %s", fa.Address).WithError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer faResp.Body.Close()

	body, err := io.ReadAll(faResp.Body)
	if err != nil {
		fa.m.AddTracef("failed to read response body from %s", fa.Address).WithError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if faResp.StatusCode < http.StatusOK || faResp.StatusCode >= http.StatusMultipleChoices {
		fa.m.AddTraceResponse("forward auth response", faResp)
		gphttp.CopyHeader(w.Header(), faResp.Header)
		gphttp.RemoveHop(w.Header())

		redirectURL, err := faResp.Location()
		if err != nil {
			fa.m.AddTracef("failed to get location from %s", fa.Address).WithError(err).WithResponse(faResp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if redirectURL.String() != "" {
			w.Header().Set("Location", redirectURL.String())
			fa.m.AddTracef("redirect to %q", redirectURL.String()).WithResponse(faResp)
		}

		w.WriteHeader(faResp.StatusCode)

		if _, err = w.Write(body); err != nil {
			fa.m.AddTracef("failed to write response body from %s", fa.Address).WithError(err).WithResponse(faResp)
		}
		return
	}

	for _, key := range fa.AuthResponseHeaders {
		key := http.CanonicalHeaderKey(key)
		req.Header.Del(key)
		if len(faResp.Header[key]) > 0 {
			req.Header[key] = append([]string(nil), faResp.Header[key]...)
		}
	}

	req.RequestURI = req.URL.RequestURI()

	authCookies := faResp.Cookies()

	if len(authCookies) == 0 {
		next.ServeHTTP(w, req)
		return
	}

	next.ServeHTTP(gphttp.NewModifyResponseWriter(w, req, func(resp *Response) error {
		fa.setAuthCookies(resp, authCookies)
		return nil
	}), req)
}

func (fa *forwardAuth) setAuthCookies(resp *Response, authCookies []*Cookie) {
	if len(fa.AddAuthCookiesToResponse) == 0 {
		return
	}

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")

	for _, cookie := range cookies {
		if !slices.Contains(fa.AddAuthCookiesToResponse, cookie.Name) {
			// this cookie is not an auth cookie, so add it back
			resp.Header.Add("Set-Cookie", cookie.String())
		}
	}

	for _, cookie := range authCookies {
		if slices.Contains(fa.AddAuthCookiesToResponse, cookie.Name) {
			// this cookie is an auth cookie, so add to resp
			resp.Header.Add("Set-Cookie", cookie.String())
		}
	}
}

func (fa *forwardAuth) setAuthHeaders(req, faReq *Request) {
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if fa.TrustForwardHeader {
			if prior, ok := req.Header[xForwardedFor]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
		}
		faReq.Header.Set(xForwardedFor, clientIP)
	}

	xMethod := req.Header.Get(xForwardedMethod)
	switch {
	case xMethod != "" && fa.TrustForwardHeader:
		faReq.Header.Set(xForwardedMethod, xMethod)
	case req.Method != "":
		faReq.Header.Set(xForwardedMethod, req.Method)
	default:
		faReq.Header.Del(xForwardedMethod)
	}

	xfp := req.Header.Get(xForwardedProto)
	switch {
	case xfp != "" && fa.TrustForwardHeader:
		faReq.Header.Set(xForwardedProto, xfp)
	case req.TLS != nil:
		faReq.Header.Set(xForwardedProto, "https")
	default:
		faReq.Header.Set(xForwardedProto, "http")
	}

	if xfp := req.Header.Get(xForwardedPort); xfp != "" && fa.TrustForwardHeader {
		faReq.Header.Set(xForwardedPort, xfp)
	}

	xfh := req.Header.Get(xForwardedHost)
	switch {
	case xfh != "" && fa.TrustForwardHeader:
		faReq.Header.Set(xForwardedHost, xfh)
	case req.Host != "":
		faReq.Header.Set(xForwardedHost, req.Host)
	default:
		faReq.Header.Del(xForwardedHost)
	}

	xfURI := req.Header.Get(xForwardedURI)
	switch {
	case xfURI != "" && fa.TrustForwardHeader:
		faReq.Header.Set(xForwardedURI, xfURI)
	case req.URL.RequestURI() != "":
		faReq.Header.Set(xForwardedURI, req.URL.RequestURI())
	default:
		faReq.Header.Del(xForwardedURI)
	}
}
