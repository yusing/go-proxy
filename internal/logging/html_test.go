package logging

import (
	"testing"

	"github.com/rs/zerolog"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestFormatHTML(t *testing.T) {
	buf := make([]byte, 0, 100)
	buf = FormatLogEntryHTML(zerolog.InfoLevel, "This is a test.\nThis is a new line.", buf)
	ExpectEqual(t, string(buf), `<pre class="log-entry">01-01 01:01 <span class="log-info">INF</span> <span class="log-message">This is a test.<br>`+prefix+`This is a new line.</span></pre>`)
}

func TestFormatHTMLANSI(t *testing.T) {
	buf := make([]byte, 0, 100)
	buf = FormatLogEntryHTML(zerolog.InfoLevel, "This is \x1b[91m\x1b[1ma test.\x1b[0mOK!.", buf)
	ExpectEqual(t, string(buf), `<pre class="log-entry">01-01 01:01 <span class="log-info">INF</span> <span class="log-message">This is <span class="log-red"><span class="log-bold">a test.</span></span>OK!.</span></pre>`)
	buf = buf[:0]
	buf = FormatLogEntryHTML(zerolog.InfoLevel, "This is \x1b[91ma \x1b[1mtest.\x1b[0mOK!.", buf)
	ExpectEqual(t, string(buf), `<pre class="log-entry">01-01 01:01 <span class="log-info">INF</span> <span class="log-message">This is <span class="log-red">a <span class="log-bold">test.</span></span>OK!.</span></pre>`)
}

func BenchmarkFormatLogEntryHTML(b *testing.B) {
	buf := make([]byte, 0, 250)
	for range b.N {
		FormatLogEntryHTML(zerolog.InfoLevel, "This is \x1b[91ma \x1b[1mtest.\x1b[0mOK!.", buf)
	}
}
