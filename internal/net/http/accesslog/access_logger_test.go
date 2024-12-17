package accesslog_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/net/http/accesslog"
	taskPkg "github.com/yusing/go-proxy/internal/task"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

type testWritter struct {
	line string
}

func (w *testWritter) Write(p []byte) (n int, err error) {
	w.line = string(p)
	return len(p), nil
}

func (w *testWritter) Close() error {
	return nil
}

var tw testWritter

const (
	remote        = "192.168.1.1"
	u             = "http://example.com/?bar=baz&foo=bar"
	uRedacted     = "http://example.com/?bar=" + RedactedValue + "&foo=" + RedactedValue
	referer       = "https://www.google.com/"
	proto         = "HTTP/1.1"
	ua            = "Go-http-client/1.1"
	status        = http.StatusOK
	contentLength = 100
	method        = http.MethodGet
)

var (
	testURL = E.Must(url.Parse(u))
	req     = &http.Request{
		RemoteAddr: remote,
		Method:     method,
		Proto:      proto,
		Host:       testURL.Host,
		URL:        testURL,
		Header: http.Header{
			"User-Agent": []string{ua},
			"Referer":    []string{referer},
			"Cookie": []string{
				"foo=bar",
				"bar=baz",
			},
		},
	}
	resp = &http.Response{
		StatusCode:    status,
		ContentLength: contentLength,
		Header:        http.Header{"Content-Type": []string{"text/plain"}},
	}
	task = taskPkg.GlobalTask("test logger")
)

func TestAccessLoggerCommon(t *testing.T) {
	config := DefaultConfig
	config.Format = FormatCommon
	logger := NewAccessLogger(task, &tw, &config)
	logger.Log(req, resp)
	ExpectEqual(t, tw.line,
		fmt.Sprintf("%s - - [%s] \"%s %s %s\" %d %d\n",
			remote, TestTimeNow, method, u, proto, status, contentLength,
		),
	)
}

func TestAccessLoggerCombined(t *testing.T) {
	config := DefaultConfig
	config.Format = FormatCombined
	logger := NewAccessLogger(task, &tw, &config)
	logger.Log(req, resp)
	ExpectEqual(t, tw.line,
		fmt.Sprintf("%s - - [%s] \"%s %s %s\" %d %d \"%s\" \"%s\"\n",
			remote, TestTimeNow, method, u, proto, status, contentLength, referer, ua,
		),
	)
}

func TestAccessLoggerRedactQuery(t *testing.T) {
	config := DefaultConfig
	config.Format = FormatCommon
	config.Fields.Query.DefaultMode = FieldModeRedact
	logger := NewAccessLogger(task, &tw, &config)
	logger.Log(req, resp)
	ExpectEqual(t, tw.line,
		fmt.Sprintf("%s - - [%s] \"%s %s %s\" %d %d\n",
			remote, TestTimeNow, method, uRedacted, proto, status, contentLength,
		),
	)
}

func getJSONEntry(t *testing.T, config *Config) JSONLogEntry {
	t.Helper()
	config.Format = FormatJSON
	logger := NewAccessLogger(task, &tw, config)
	logger.Log(req, resp)
	var entry JSONLogEntry
	err := json.Unmarshal([]byte(tw.line), &entry)
	ExpectNoError(t, err)
	return entry
}

func TestAccessLoggerJSON(t *testing.T) {
	config := DefaultConfig
	entry := getJSONEntry(t, &config)
	ExpectEqual(t, entry.IP, remote)
	ExpectEqual(t, entry.Method, method)
	ExpectEqual(t, entry.Scheme, "http")
	ExpectEqual(t, entry.Host, testURL.Host)
	ExpectEqual(t, entry.URI, testURL.RequestURI())
	ExpectEqual(t, entry.Protocol, proto)
	ExpectEqual(t, entry.Status, status)
	ExpectEqual(t, entry.ContentType, "text/plain")
	ExpectEqual(t, entry.Size, contentLength)
	ExpectEqual(t, entry.Referer, referer)
	ExpectEqual(t, entry.UserAgent, ua)
	ExpectEqual(t, len(entry.Headers), 0)
	ExpectEqual(t, len(entry.Cookies), 0)
}
