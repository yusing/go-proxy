// Modified from Traefik Labs's MIT-licensed code (https://github.com/traefik/traefik/blob/master/pkg/middlewares/auth/forward.go)
// Copyright (c) 2020-2024 Traefik Labs
// Copyright (c) 2024 yusing

package middleware

import (
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

type (
	forwardAuth struct {
		forwardAuthOpts
		m *Middleware
	}
	forwardAuthOpts struct {
		Address                  string `validate:"url,required"`
		TrustForwardHeader       bool
		AuthResponseHeaders      []string
		AddAuthCookiesToResponse []string
	}
)

var ForwardAuth = &Middleware{withOptions: NewForwardAuth}

var faHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(r *Request, via []*Request) error {
		return http.ErrUseLastResponse
	},
}

func NewForwardAuth(optsRaw OptionsRaw) (*Middleware, E.Error) {
	fa := new(forwardAuth)
	if err := Deserialize(optsRaw, &fa.forwardAuthOpts); err != nil {
		return nil, err
	}
	fa.m = &Middleware{
		impl:   fa,
		before: fa.forward,
	}
	return fa.m, nil
}

func (fa *forwardAuth) forward(next http.HandlerFunc, w ResponseWriter, req *Request) {
	gphttp.RemoveHop(req.Header)

	// Construct original URL for the redirect
	// scheme := "http"
	// if req.TLS != nil {
	// 	scheme = "https"
	// }
	// originalURL := scheme + "://" + req.Host + req.RequestURI

	url := fa.Address
	faReq, err := http.NewRequestWithContext(
		req.Context(),
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		fa.m.AddTracef("new request err to %s", url).WithError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gphttp.CopyHeader(faReq.Header, req.Header)
	gphttp.RemoveHop(faReq.Header)

	faReq.Header = gphttp.FilterHeaders(faReq.Header, fa.AuthResponseHeaders)
	fa.setAuthHeaders(req, faReq)
	// Set headers needed by Authentik
	// faReq.Header.Set("X-Original-URL", originalURL)
	fa.m.AddTraceRequest("forward auth request", faReq)

	faResp, err := faHTTPClient.Do(faReq)
	if err != nil {
		fa.m.AddTracef("failed to call %s", url).WithError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer faResp.Body.Close()

	body, err := io.ReadAll(faResp.Body)
	if err != nil {
		fa.m.AddTracef("failed to read response body from %s", url).WithError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if faResp.StatusCode < http.StatusOK || faResp.StatusCode >= http.StatusMultipleChoices {
		fa.m.AddTraceResponse("forward auth response", faResp)
		gphttp.CopyHeader(w.Header(), faResp.Header)
		gphttp.RemoveHop(w.Header())

		redirectURL, err := faResp.Location()
		if err != nil {
			fa.m.AddTracef("failed to get location from %s", url).WithError(err).WithResponse(faResp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if redirectURL.String() != "" {
			w.Header().Set("Location", redirectURL.String())
			fa.m.AddTracef("%s", "redirect to "+redirectURL.String())
		}

		w.WriteHeader(faResp.StatusCode)

		if _, err = w.Write(body); err != nil {
			fa.m.AddTracef("failed to write response body from %s", url).WithError(err).WithResponse(faResp)
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

	next.ServeHTTP(gphttp.NewModifyResponseWriter(w, req, func(resp *http.Response) error {
		fa.setAuthCookies(resp, authCookies)
		return nil
	}), req)
}

func (fa *forwardAuth) setAuthCookies(resp *http.Response, authCookies []*Cookie) {
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
			if prior, ok := req.Header[gphttp.HeaderXForwardedFor]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
		}
		faReq.Header.Set(gphttp.HeaderXForwardedFor, clientIP)
	}

	xMethod := req.Header.Get(gphttp.HeaderXForwardedMethod)
	switch {
	case xMethod != "" && fa.TrustForwardHeader:
		faReq.Header.Set(gphttp.HeaderXForwardedMethod, xMethod)
	case req.Method != "":
		faReq.Header.Set(gphttp.HeaderXForwardedMethod, req.Method)
	default:
		faReq.Header.Del(gphttp.HeaderXForwardedMethod)
	}

	xfp := req.Header.Get(gphttp.HeaderXForwardedProto)
	switch {
	case xfp != "" && fa.TrustForwardHeader:
		faReq.Header.Set(gphttp.HeaderXForwardedProto, xfp)
	case req.TLS != nil:
		faReq.Header.Set(gphttp.HeaderXForwardedProto, "https")
	default:
		faReq.Header.Set(gphttp.HeaderXForwardedProto, "http")
	}

	if xfp := req.Header.Get(gphttp.HeaderXForwardedPort); xfp != "" && fa.TrustForwardHeader {
		faReq.Header.Set(gphttp.HeaderXForwardedPort, xfp)
	}

	xfh := req.Header.Get(gphttp.HeaderXForwardedHost)
	switch {
	case xfh != "" && fa.TrustForwardHeader:
		faReq.Header.Set(gphttp.HeaderXForwardedHost, xfh)
	case req.Host != "":
		faReq.Header.Set(gphttp.HeaderXForwardedHost, req.Host)
	default:
		faReq.Header.Del(gphttp.HeaderXForwardedHost)
	}

	xfURI := req.Header.Get(gphttp.HeaderXForwardedURI)
	switch {
	case xfURI != "" && fa.TrustForwardHeader:
		faReq.Header.Set(gphttp.HeaderXForwardedURI, xfURI)
	case req.URL.RequestURI() != "":
		faReq.Header.Set(gphttp.HeaderXForwardedURI, req.URL.RequestURI())
	default:
		faReq.Header.Del(gphttp.HeaderXForwardedURI)
	}
}
