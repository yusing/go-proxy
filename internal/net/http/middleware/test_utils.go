package middleware

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/types"
)

//go:embed test_data/sample_headers.json
var testHeadersRaw []byte
var testHeaders http.Header

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
	reqURL        types.URL
	upstreamURL   types.URL
	body          []byte
	realRoundTrip bool
	headers       http.Header
}

func newMiddlewareTest(middleware *Middleware, args *testArgs) (*TestResult, E.Error) {
	var body io.Reader
	var rr requestRecorder
	var err error

	if args == nil {
		args = new(testArgs)
	}

	if args.body != nil {
		body = bytes.NewReader(args.body)
	}

	if args.reqURL.Nil() {
		args.reqURL = E.Must(types.ParseURL("https://example.com"))
	}

	req := httptest.NewRequest(http.MethodGet, args.reqURL.String(), body)
	for k, v := range args.headers {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()

	if args.upstreamURL.Nil() {
		args.upstreamURL = E.Must(types.ParseURL("https://10.0.0.1:8443")) // dummy url, no actual effect
	}

	if args.realRoundTrip {
		rr.parent = http.DefaultTransport
	}
	rp := gphttp.NewReverseProxy(middleware.name, args.upstreamURL, &rr)
	mid, setOptErr := middleware.WithOptionsClone(args.middlewareOpt)
	if setOptErr != nil {
		return nil, setOptErr
	}
	patchReverseProxy(rp, []*Middleware{mid})
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
