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

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/common"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	gpHTTP "github.com/yusing/go-proxy/http"
	U "github.com/yusing/go-proxy/utils"
)

type (
	forwardAuth struct {
		*forwardAuthOpts
		m      *Middleware
		client http.Client
	}
	forwardAuthOpts struct {
		Address                  string
		TrustForwardHeader       bool
		AuthResponseHeaders      []string
		AddAuthCookiesToResponse []string
	}
)

const (
	xForwardedFor    = "X-Forwarded-For"
	xForwardedMethod = "X-Forwarded-Method"
	xForwardedHost   = "X-Forwarded-Host"
	xForwardedProto  = "X-Forwarded-Proto"
	xForwardedURI    = "X-Forwarded-Uri"
	xForwardedPort   = "X-Forwarded-Port"
)

var ForwardAuth = newForwardAuth()
var faLogger = logrus.WithField("middleware", "ForwardAuth")

func newForwardAuth() (fa *forwardAuth) {
	fa = new(forwardAuth)
	fa.m = new(Middleware)
	fa.m.labelParserMap = D.ValueParserMap{
		"trust_forward_header":         D.BoolParser,
		"auth_response_headers":        D.YamlStringListParser,
		"add_auth_cookies_to_response": D.YamlStringListParser,
	}
	fa.m.withOptions = func(optsRaw OptionsRaw, rp *ReverseProxy) (*Middleware, E.NestedError) {
		tr, ok := rp.Transport.(*http.Transport)
		if ok {
			tr = tr.Clone()
		} else {
			tr = common.DefaultTransport.Clone()
		}

		faWithOpts := new(forwardAuth)
		faWithOpts.forwardAuthOpts = new(forwardAuthOpts)
		faWithOpts.client = http.Client{
			CheckRedirect: func(r *Request, via []*Request) error {
				return http.ErrUseLastResponse
			},
			Timeout:   30 * time.Second,
			Transport: tr,
		}
		faWithOpts.m = &Middleware{
			impl:   faWithOpts,
			before: fa.forward,
		}

		err := U.Deserialize(optsRaw, faWithOpts.forwardAuthOpts)
		if err != nil {
			return nil, E.FailWith("set options", err)
		}
		_, err = E.Check(url.Parse(faWithOpts.Address))
		if err != nil {
			return nil, E.Invalid("address", faWithOpts.Address)
		}
		return faWithOpts.m, nil
	}
	return
}

func (fa *forwardAuth) forward(next http.Handler, w ResponseWriter, req *Request) {
	removeHop(req.Header)

	faReq, err := http.NewRequestWithContext(
		req.Context(),
		http.MethodGet,
		fa.Address,
		nil,
	)
	if err != nil {
		faLogger.Debugf("new request err to %s: %s", fa.Address, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	copyHeader(faReq.Header, req.Header)
	removeHop(faReq.Header)

	filterHeaders(faReq.Header, fa.AuthResponseHeaders)
	fa.setAuthHeaders(req, faReq)

	faResp, err := fa.client.Do(faReq)
	if err != nil {
		faLogger.Debugf("failed to call %s: %s", fa.Address, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer faResp.Body.Close()

	body, err := io.ReadAll(faResp.Body)
	if err != nil {
		faLogger.Debugf("failed to read response body from %s: %s", fa.Address, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if faResp.StatusCode < http.StatusOK || faResp.StatusCode >= http.StatusMultipleChoices {
		copyHeader(w.Header(), faResp.Header)
		removeHop(w.Header())

		redirectURL, err := faResp.Location()
		if err != nil {
			faLogger.Debugf("failed to get location from %s: %s", fa.Address, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if redirectURL.String() != "" {
			w.Header().Set("Location", redirectURL.String())
		}

		w.WriteHeader(faResp.StatusCode)

		if _, err = w.Write(body); err != nil {
			faLogger.Debugf("failed to write response body from %s: %s", fa.Address, err)
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

	next.ServeHTTP(gpHTTP.NewModifyResponseWriter(w, req, func(resp *Response) error {
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
