package accesslog_test

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/net/http/accesslog"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

// Cookie header should be removed,
// stored in JSONLogEntry.Cookies instead.
func TestAccessLoggerJSONKeepHeaders(t *testing.T) {
	config := DefaultConfig()
	config.Fields.Headers.DefaultMode = FieldModeKeep
	entry := getJSONEntry(t, config)
	ExpectDeepEqual(t, len(entry.Headers["Cookie"]), 0)
	for k, v := range req.Header {
		if k != "Cookie" {
			ExpectDeepEqual(t, entry.Headers[k], v)
		}
	}
}

func TestAccessLoggerJSONRedactHeaders(t *testing.T) {
	config := DefaultConfig()
	config.Fields.Headers.DefaultMode = FieldModeRedact
	entry := getJSONEntry(t, config)
	ExpectDeepEqual(t, len(entry.Headers["Cookie"]), 0)
	for k := range req.Header {
		if k != "Cookie" {
			ExpectDeepEqual(t, entry.Headers[k], []string{RedactedValue})
		}
	}
}

func TestAccessLoggerJSONKeepCookies(t *testing.T) {
	config := DefaultConfig()
	config.Fields.Headers.DefaultMode = FieldModeKeep
	config.Fields.Cookies.DefaultMode = FieldModeKeep
	entry := getJSONEntry(t, config)
	ExpectDeepEqual(t, len(entry.Headers["Cookie"]), 0)
	for _, cookie := range req.Cookies() {
		ExpectEqual(t, entry.Cookies[cookie.Name], cookie.Value)
	}
}

func TestAccessLoggerJSONRedactCookies(t *testing.T) {
	config := DefaultConfig()
	config.Fields.Headers.DefaultMode = FieldModeKeep
	config.Fields.Cookies.DefaultMode = FieldModeRedact
	entry := getJSONEntry(t, config)
	ExpectDeepEqual(t, len(entry.Headers["Cookie"]), 0)
	for _, cookie := range req.Cookies() {
		ExpectEqual(t, entry.Cookies[cookie.Name], RedactedValue)
	}
}

func TestAccessLoggerJSONDropQuery(t *testing.T) {
	config := DefaultConfig()
	config.Fields.Query.DefaultMode = FieldModeDrop
	entry := getJSONEntry(t, config)
	ExpectDeepEqual(t, entry.Query["foo"], nil)
	ExpectDeepEqual(t, entry.Query["bar"], nil)
}

func TestAccessLoggerJSONRedactQuery(t *testing.T) {
	config := DefaultConfig()
	config.Fields.Query.DefaultMode = FieldModeRedact
	entry := getJSONEntry(t, config)
	ExpectDeepEqual(t, entry.Query["foo"], []string{RedactedValue})
	ExpectDeepEqual(t, entry.Query["bar"], []string{RedactedValue})
}
