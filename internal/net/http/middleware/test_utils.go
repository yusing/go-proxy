package middleware

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	E "github.com/yusing/go-proxy/internal/error"
	gpHTTP "github.com/yusing/go-proxy/internal/net/http"
)

//go:embed test_data/sample_headers.json
var testHeadersRaw []byte
var testHeaders http.Header

const testHost = "example.com"

func init() {
	tmp := map[string]string{}
	err := json.Unmarshal(testHeadersRaw, &tmp)
	if err != nil {
		panic(err)
	}
	testHeaders = http.Header{}
	for k, v := range tmp {
		testHeaders.Set(k, v)
	}
}

type requestHeaderRecorder struct {
	parent     http.RoundTripper
	reqHeaders http.Header
}

func (rt *requestHeaderRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.reqHeaders = req.Header
	if rt.parent != nil {
		return rt.parent.RoundTrip(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     testHeaders,
		Body:       io.NopCloser(bytes.NewBufferString("OK")),
		Request:    req,
	}, nil
}

type TestResult struct {
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	ResponseStatus  int
	Data            []byte
}

type testArgs struct {
	middlewareOpt OptionsRaw
	proxyURL      string
	body          []byte
	scheme        string
}

func newMiddlewareTest(middleware *Middleware, args *testArgs) (*TestResult, E.NestedError) {
	var body io.Reader
	var rt = new(requestHeaderRecorder)
	var proxyURL *url.URL
	var requestTarget string
	var err error

	if args == nil {
		args = new(testArgs)
	}

	if args.body != nil {
		body = bytes.NewReader(args.body)
	}

	if args.scheme == "" || args.scheme == "http" {
		requestTarget = "http://" + testHost
	} else if args.scheme == "https" {
		requestTarget = "https://" + testHost
	} else {
		panic("typo?")
	}

	req := httptest.NewRequest(http.MethodGet, requestTarget, body)
	w := httptest.NewRecorder()

	if args.scheme == "https" && req.TLS == nil {
		panic("bug occurred")
	}

	if args.proxyURL != "" {
		proxyURL, err = url.Parse(args.proxyURL)
		if err != nil {
			return nil, E.From(err)
		}
		rt.parent = http.DefaultTransport
	} else {
		proxyURL, _ = url.Parse("https://" + testHost) // dummy url, no actual effect
	}
	rp := gpHTTP.NewReverseProxy(proxyURL, rt)
	setOptErr := PatchReverseProxy(rp, map[string]OptionsRaw{
		middleware.name: args.middlewareOpt,
	})
	if setOptErr != nil {
		return nil, setOptErr
	}
	rp.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, E.From(err)
	}
	return &TestResult{
		RequestHeaders:  rt.reqHeaders,
		ResponseHeaders: resp.Header,
		ResponseStatus:  resp.StatusCode,
		Data:            data,
	}, nil
}
