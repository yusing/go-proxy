package logging

import (
	"bytes"
	"time"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	ansiPkg "github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

func fmtMessageToHTMLBytes(msg string, buf []byte) []byte {
	buf = append(buf, []byte(`<span class="log-message">`)...)
	var last byte

	isAnsi := false
	nAnsi := 0
	ansi := bytes.NewBuffer(make([]byte, 0, 4))
	ansiContent := bytes.NewBuffer(make([]byte, 0, 30))
	style := bytes.NewBuffer(make([]byte, 0, 30))

	for _, r := range msg {
		if last == '\n' {
			buf = append(buf, prefixHTML...)
		}
		if last == '\x1b' {
			if r != 'm' {
				ansi.WriteRune(r)
				if r == '[' && ansiContent.Len() > 0 {
					buf = append(buf, []byte(`<span `)...)
					buf = append(buf, style.Bytes()...)
					buf = append(buf, []byte(`>`)...)
					buf = append(buf, ansiContent.Bytes()...)
					style.Reset()
					ansiContent.Reset()
					nAnsi++
				}
			} else {
				ansiCode := ansi.String()
				switch ansiCode {
				case "[0": // reset
					if style.Len() > 0 {
						buf = append(buf, []byte(`<span `)...)
						buf = append(buf, style.Bytes()...)
						buf = append(buf, []byte(`>`)...)
					}
					for nAnsi-1 > 0 {
						buf = append(buf, []byte(`</span>`)...)
						nAnsi--
					}
					nAnsi = 0
					buf = append(buf, ansiContent.Bytes()...)
					buf = append(buf, []byte(`</span>`)...)
					isAnsi = false
					ansiContent.Reset()
					style.Reset()
				case "[1": // bold
					style.WriteString(`class="log-bold" `)
				default:
					className, ok := ansiPkg.ToHTMLClass[ansiCode]
					if ok {
						style.WriteString(`class="` + className + `" `)
					} else {
						style.WriteString(`class="log-unknown-ansi" `)
					}
				}
				ansi.Reset()
				last = 0
			}
			continue
		}

		last = byte(r)
		if r == '\x1b' {
			isAnsi = true
			continue
		}
		if isAnsi || nAnsi > 0 {
			if symbol, ok := symbolMapping[r]; ok {
				ansiContent.Write(symbol)
			} else {
				ansiContent.WriteRune(r)
			}
		} else {
			if symbol, ok := symbolMapping[r]; ok {
				buf = append(buf, symbol...)
			} else {
				buf = append(buf, last)
			}
		}
	}

	buf = append(buf, []byte("</span>")...)
	return buf
}

var levelHTMLFormats = [][]byte{
	[]byte(` <span class="log-trace">TRC</span> `),
	[]byte(` <span class="log-debug">DBG</span> `),
	[]byte(` <span class="log-info">INF</span> `),
	[]byte(` <span class="log-warn">WRN</span> `),
	[]byte(` <span class="log-error">ERR</span> `),
	[]byte(` <span class="log-fatal">FTL</span> `),
	[]byte(` <span class="log-panic">PAN</span> `),
}

var symbolMapping = map[rune][]byte{
	'•':  []byte("&middot;"),
	'>':  []byte("&gt;"),
	'<':  []byte("&lt;"),
	'\t': []byte("&ensp;"),
	'\n': []byte("<br>"),
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
	buf = fmtMessageToHTMLBytes(message, buf)
	buf = append(buf, []byte("</pre>")...)
	return buf
}
