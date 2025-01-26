package logging

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
)

var levelHTMLFormats = [][]byte{
	[]byte(` <span class="log-trace">TRC</span> `),
	[]byte(` <span class="log-debug">DBG</span> `),
	[]byte(` <span class="log-info">INF</span> `),
	[]byte(` <span class="log-warn">WRN</span> `),
	[]byte(` <span class="log-error">ERR</span> `),
	[]byte(` <span class="log-fatal">FTL</span> `),
	[]byte(` <span class="log-panic">PAN</span> `),
}

var colorToClass = map[string]string{
	"1":  "log-bold",
	"3":  "log-italic",
	"4":  "log-underline",
	"30": "log-black",
	"31": "log-red",
	"32": "log-green",
	"33": "log-yellow",
	"34": "log-blue",
	"35": "log-magenta",
	"36": "log-cyan",
	"37": "log-white",
	"90": "log-bright-black",
	"91": "log-red",
	"92": "log-bright-green",
	"93": "log-bright-yellow",
	"94": "log-bright-blue",
	"95": "log-bright-magenta",
	"96": "log-bright-cyan",
	"97": "log-bright-white",
}

// FormatMessageToHTMLBytes converts text with ANSI color codes to HTML with class names.
// ANSI codes are mapped to classes via a static map, and reset codes ([0m) close all spans.
// Time complexity is O(n) with minimal allocations.
func FormatMessageToHTMLBytes(msg string, buf []byte) ([]byte, error) {
	buf = append(buf, "<span class=\"log-message\">"...)
	var stack []string
	lastPos := 0

	for i := 0; i < len(msg); {
		if msg[i] == '\x1b' && i+1 < len(msg) && msg[i+1] == '[' {
			if lastPos < i {
				escapeAndAppend(msg[lastPos:i], &buf)
			}
			i += 2 // Skip \x1b[

			start := i
			for ; i < len(msg) && msg[i] != 'm'; i++ {
				if !isANSICodeChar(msg[i]) {
					return nil, fmt.Errorf("invalid ANSI char: %c", msg[i])
				}
			}

			if i >= len(msg) {
				return nil, errors.New("unterminated ANSI sequence")
			}

			codeStr := msg[start:i]
			i++ // Skip 'm'
			lastPos = i

			startPart := 0
			for j := 0; j <= len(codeStr); j++ {
				if j == len(codeStr) || codeStr[j] == ';' {
					part := codeStr[startPart:j]
					if part == "" {
						return nil, errors.New("empty code part")
					}

					if part == "0" {
						for range stack {
							buf = append(buf, "</span>"...)
						}
						stack = stack[:0]
					} else {
						className, ok := colorToClass[part]
						if !ok {
							return nil, fmt.Errorf("invalid ANSI code: %s", part)
						}
						stack = append(stack, className)
						buf = append(buf, `<span class="`...)
						buf = append(buf, className...)
						buf = append(buf, `">`...)
					}
					startPart = j + 1
				}
			}
		} else {
			i++
		}
	}

	if lastPos < len(msg) {
		escapeAndAppend(msg[lastPos:], &buf)
	}

	for range stack {
		buf = append(buf, "</span>"...)
	}

	buf = append(buf, "</span>"...)
	return buf, nil
}

func isANSICodeChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == ';'
}

func escapeAndAppend(s string, buf *[]byte) {
	for i, r := range s {
		switch r {
		case '•':
			*buf = append(*buf, "&middot;"...)
		case '&':
			*buf = append(*buf, "&amp;"...)
		case '<':
			*buf = append(*buf, "&lt;"...)
		case '>':
			*buf = append(*buf, "&gt;"...)
		case '\t':
			*buf = append(*buf, "&#9;"...)
		case '\n':
			*buf = append(*buf, "<br>"...)
		default:
			*buf = append(*buf, s[i])
		}
	}
}

func timeNowHTML() []byte {
	if !common.IsTest {
		return []byte(time.Now().Format(timeFmt))
	}
	return []byte(time.Date(2024, 1, 1, 1, 1, 1, 1, time.UTC).Format(timeFmt))
}

func FormatLogEntryHTML(level zerolog.Level, message string, buf []byte) []byte {
	buf = append(buf, []byte(`<pre class="log-entry">`)...)
	buf = append(buf, timeNowHTML()...)
	if level < zerolog.NoLevel {
		buf = append(buf, levelHTMLFormats[level+1]...)
	}
	buf, _ = FormatMessageToHTMLBytes(message, buf)
	buf = append(buf, []byte("</pre>")...)
	return buf
}
