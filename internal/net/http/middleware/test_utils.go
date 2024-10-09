package middleware

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/types"
)

//go:embed test_data/sample_headers.json
var testHeadersRaw []byte
var testHeaders http.Header

const testHost = "example.com"

func init() {
	if !common.IsTest {
		return
	}
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

type requestRecorder struct {
	parent     http.RoundTripper
	headers    http.Header
	remoteAddr string
}

func (rt *requestRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.headers = req.Header
	rt.remoteAddr = req.RemoteAddr
	if rt.parent != nil {
		return rt.parent.RoundTrip(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     testHeaders,
		Body:       io.NopCloser(bytes.NewBufferString("OK")),
		Request:    req,
		TLS:        req.TLS,
	}, nil
}

type TestResult struct {
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	ResponseStatus  int
	RemoteAddr      string
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
	var rr = new(requestRecorder)
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
		rr.parent = http.DefaultTransport
	} else {
		proxyURL, _ = url.Parse("https://" + testHost) // dummy url, no actual effect
	}
	rp := gphttp.NewReverseProxy(types.NewURL(proxyURL), rr)
	mid, setOptErr := middleware.WithOptionsClone(args.middlewareOpt)
	if setOptErr != nil {
		return nil, setOptErr
	}
	patchReverseProxy(middleware.name, rp, []*Middleware{mid})
	rp.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, E.From(err)
	}
	return &TestResult{
		RequestHeaders:  rr.headers,
		ResponseHeaders: resp.Header,
		ResponseStatus:  resp.StatusCode,
		RemoteAddr:      rr.remoteAddr,
		Data:            data,
	}, nil
}
