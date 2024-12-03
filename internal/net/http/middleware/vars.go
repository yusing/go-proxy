package middleware

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

type varReplaceFunc func(string) string

var (
	reArg        = regexp.MustCompile(`\$arg\([\w-_]+\)`)
	reHeader     = regexp.MustCompile(`\$header\([\w-]+\)`)
	reRespHeader = regexp.MustCompile(`\$resp_header\([\w-]+\)`)
	reStatic     = regexp.MustCompile(`\$[\w_]+`)
)

func varSubsMap(req *Request, resp *Response) map[string]func() string {
	reqHost, reqPort, err := net.SplitHostPort(req.Host)
	if err != nil {
		reqHost = req.Host
	}
	reqAddr := reqHost
	if reqPort != "" {
		reqAddr += ":" + reqPort
	}

	pairs := map[string]func() string{
		"$req_method":         func() string { return req.Method },
		"$req_scheme":         func() string { return req.URL.Scheme },
		"$req_host":           func() string { return reqHost },
		"$req_port":           func() string { return reqPort },
		"$req_addr":           func() string { return reqAddr },
		"$req_path":           func() string { return req.URL.Path },
		"$req_query":          func() string { return req.URL.RawQuery },
		"$req_url":            func() string { return req.URL.String() },
		"$req_uri":            req.URL.RequestURI,
		"$req_content_type":   func() string { return req.Header.Get("Content-Type") },
		"$req_content_length": func() string { return strconv.FormatInt(req.ContentLength, 10) },
		"$remote_addr":        func() string { return req.RemoteAddr },
	}

	if resp != nil {
		pairs["$resp_content_type"] = func() string { return resp.Header.Get("Content-Type") }
		pairs["$resp_content_length"] = func() string { return resp.Header.Get("Content-Length") }
		pairs["$status_code"] = func() string { return strconv.Itoa(resp.StatusCode) }
	}

	upScheme := req.Header.Get(gphttp.HeaderUpstreamScheme)
	if upScheme == "" {
		return pairs
	}

	upHost := req.Header.Get(gphttp.HeaderUpstreamHost)
	upPort := req.Header.Get(gphttp.HeaderUpstreamPort)
	upAddr := upHost
	if upPort != "" {
		upAddr += ":" + upPort
	}
	upURL := upScheme + "://" + upAddr

	pairs["$upstream_scheme"] = func() string { return upScheme }
	pairs["$upstream_host"] = func() string { return upHost }
	pairs["$upstream_port"] = func() string { return upPort }
	pairs["$upstream_addr"] = func() string { return upAddr }
	pairs["$upstream_url"] = func() string { return upURL }

	return pairs
}

func varReplacer(req *Request, resp *Response) varReplaceFunc {
	pairs := varSubsMap(req, resp)
	return func(s string) string {
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

		// Replace headers
		s = reHeader.ReplaceAllStringFunc(s, func(match string) string {
			header := http.CanonicalHeaderKey(match[8 : len(match)-1])
			return req.Header.Get(header)
		})

		if resp != nil {
			s = reRespHeader.ReplaceAllStringFunc(s, func(match string) string {
				header := http.CanonicalHeaderKey(match[14 : len(match)-1])
				return resp.Header.Get(header)
			})
		}

		// Replace static variables
		return reStatic.ReplaceAllStringFunc(s, func(match string) string {
			if fn, ok := pairs[match]; ok {
				return fn()
			}
			return match
		})
	}
}

func varReplacerDummy(s string) string {
	return s
}
