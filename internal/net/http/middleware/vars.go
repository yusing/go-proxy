package middleware

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

type (
	reqVarGetter  func(*Request) string
	respVarGetter func(*Response) string
)

var (
	reArg        = regexp.MustCompile(`\$arg\([\w-_]+\)`)
	reReqHeader  = regexp.MustCompile(`\$header\([\w-]+\)`)
	reRespHeader = regexp.MustCompile(`\$resp_header\([\w-]+\)`)
	reStatic     = regexp.MustCompile(`\$[\w_]+`)
)

var staticReqVarSubsMap = map[string]reqVarGetter{
	"$req_method": func(req *Request) string { return req.Method },
	"$req_scheme": func(req *Request) string { return req.URL.Scheme },
	"$req_host": func(req *Request) string {
		reqHost, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			return req.Host
		}
		return reqHost
	},
	"$req_port": func(req *Request) string {
		_, reqPort, _ := net.SplitHostPort(req.Host)
		return reqPort
	},
	"$req_addr":           func(req *Request) string { return req.Host },
	"$req_path":           func(req *Request) string { return req.URL.Path },
	"$req_query":          func(req *Request) string { return req.URL.RawQuery },
	"$req_url":            func(req *Request) string { return req.URL.String() },
	"$req_uri":            func(req *Request) string { return req.URL.RequestURI() },
	"$req_content_type":   func(req *Request) string { return req.Header.Get("Content-Type") },
	"$req_content_length": func(req *Request) string { return strconv.FormatInt(req.ContentLength, 10) },
	"$remote_addr":        func(req *Request) string { return req.RemoteAddr },
	"$upstream_scheme":    func(req *Request) string { return req.Header.Get(gphttp.HeaderUpstreamScheme) },
	"$upstream_host":      func(req *Request) string { return req.Header.Get(gphttp.HeaderUpstreamHost) },
	"$upstream_port":      func(req *Request) string { return req.Header.Get(gphttp.HeaderUpstreamPort) },
	"$upstream_addr": func(req *Request) string {
		upHost := req.Header.Get(gphttp.HeaderUpstreamHost)
		upPort := req.Header.Get(gphttp.HeaderUpstreamPort)
		if upPort != "" {
			return upHost + ":" + upPort
		}
		return upHost
	},
	"$upstream_url": func(req *Request) string {
		upScheme := req.Header.Get(gphttp.HeaderUpstreamScheme)
		if upScheme == "" {
			return ""
		}
		upHost := req.Header.Get(gphttp.HeaderUpstreamHost)
		upPort := req.Header.Get(gphttp.HeaderUpstreamPort)
		upAddr := upHost
		if upPort != "" {
			upAddr += ":" + upPort
		}
		return upScheme + "://" + upAddr
	},
}

var staticRespVarSubsMap = map[string]respVarGetter{
	"$resp_content_type":   func(resp *Response) string { return resp.Header.Get("Content-Type") },
	"$resp_content_length": func(resp *Response) string { return resp.Header.Get("Content-Length") },
	"$status_code":         func(resp *Response) string { return strconv.Itoa(resp.StatusCode) },
}

func varReplace(req *Request, resp *Response, s string) string {
	if req != nil {
		// Replace query parameters
		s = reArg.ReplaceAllStringFunc(s, func(match string) string {
			name := match[5 : len(match)-1]
			for k, v := range req.URL.Query() {
				if strings.EqualFold(k, name) {
					return v[0]
				}
			}
			return ""
		})

		// Replace request headers
		s = reReqHeader.ReplaceAllStringFunc(s, func(match string) string {
			header := http.CanonicalHeaderKey(match[8 : len(match)-1])
			return req.Header.Get(header)
		})
	}

	if resp != nil {
		// Replace response headers
		s = reRespHeader.ReplaceAllStringFunc(s, func(match string) string {
			header := http.CanonicalHeaderKey(match[14 : len(match)-1])
			return resp.Header.Get(header)
		})
	}

	// Replace static variables
	if req != nil {
		s = reStatic.ReplaceAllStringFunc(s, func(match string) string {
			if fn, ok := staticReqVarSubsMap[match]; ok {
				return fn(req)
			}
			return match
		})
	}

	if resp != nil {
		s = reStatic.ReplaceAllStringFunc(s, func(match string) string {
			if fn, ok := staticRespVarSubsMap[match]; ok {
				return fn(resp)
			}
			return match
		})
	}

	return s
}
