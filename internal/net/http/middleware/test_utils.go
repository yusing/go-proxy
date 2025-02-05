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
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
	"github.com/yusing/go-proxy/internal/net/types"
	. "github.com/yusing/go-proxy/internal/utils/testing"
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
	args *testArgs

	parent     http.RoundTripper
	headers    http.Header
	remoteAddr string
}

func newRequestRecorder(args *testArgs) *requestRecorder {
	return &requestRecorder{args: args}
}

func (rt *requestRecorder) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	rt.headers = req.Header
	rt.remoteAddr = req.RemoteAddr
	if rt.parent != nil {
		resp, err = rt.parent.RoundTrip(req)
	} else {
		resp = &http.Response{
			Status:        http.StatusText(rt.args.respStatus),
			StatusCode:    rt.args.respStatus,
			Header:        testHeaders,
			Body:          io.NopCloser(bytes.NewReader(rt.args.respBody)),
			ContentLength: int64(len(rt.args.respBody)),
			Request:       req,
			TLS:           req.TLS,
		}
	}
	if err == nil {
		for k, v := range rt.args.respHeaders {
			resp.Header[k] = v
		}
	}
	return resp, nil
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
	upstreamURL   *types.URL

	realRoundTrip bool

	reqURL    *types.URL
	reqMethod string
	headers   http.Header
	body      []byte

	respHeaders http.Header
	respBody    []byte
	respStatus  int
}

func (args *testArgs) setDefaults() {
	if args.reqURL == nil {
		args.reqURL = Must(types.ParseURL("https://example.com"))
	}
	if args.reqMethod == "" {
		args.reqMethod = http.MethodGet
	}
	if args.upstreamURL == nil {
		args.upstreamURL = Must(types.ParseURL("https://10.0.0.1:8443")) // dummy url, no actual effect
	}
	if args.respHeaders == nil {
		args.respHeaders = http.Header{}
	}
	if args.respBody == nil {
		args.respBody = []byte("OK")
	}
	if args.respStatus == 0 {
		args.respStatus = http.StatusOK
	}
}

func (args *testArgs) bodyReader() io.Reader {
	if args.body != nil {
		return bytes.NewReader(args.body)
	}
	return nil
}

func newMiddlewareTest(middleware *Middleware, args *testArgs) (*TestResult, E.Error) {
	if args == nil {
		args = new(testArgs)
	}
	args.setDefaults()

	mid, setOptErr := middleware.New(args.middlewareOpt)
	if setOptErr != nil {
		return nil, setOptErr
	}

	return newMiddlewaresTest([]*Middleware{mid}, args)
}

func newMiddlewaresTest(middlewares []*Middleware, args *testArgs) (*TestResult, E.Error) {
	if args == nil {
		args = new(testArgs)
	}
	args.setDefaults()

	req := httptest.NewRequest(args.reqMethod, args.reqURL.String(), args.bodyReader())
	for k, v := range args.headers {
		req.Header[k] = v
	}

	w := httptest.NewRecorder()

	rr := newRequestRecorder(args)
	if args.realRoundTrip {
		rr.parent = http.DefaultTransport
	}

	rp := reverseproxy.NewReverseProxy("test", args.upstreamURL, rr)
	patchReverseProxy(rp, middlewares)
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
