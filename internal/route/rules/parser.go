package rules

import (
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

var escapedChars = map[rune]rune{
	'n':  '\n',
	't':  '\t',
	'r':  '\r',
	'\'': '\'',
	'"':  '"',
	'\\': '\\',
	' ':  ' ',
}

// parse expression to subject and args
// with support for quotes and escaped chars, e.g.
//
//	error 403 "Forbidden 'foo' 'bar'"
//	error 403 Forbidden\ \"foo\"\ \"bar\".
func parse(v string) (subject string, args []string, err E.Error) {
	v = strings.TrimSpace(v)
	var buf strings.Builder
	escaped := false
	quotes := make([]rune, 0, 4)
	flush := func() {
		if subject == "" {
			subject = buf.String()
		} else {
			args = append(args, buf.String())
		}
		buf.Reset()
	}
	for _, r := range v {
		if escaped {
			if ch, ok := escapedChars[r]; ok {
				buf.WriteRune(ch)
			} else {
				err = ErrUnsupportedEscapeChar.Subjectf("\\%c", r)
				return
			}
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
			continue
		case '"', '\'':
			switch {
			case len(quotes) > 0 && quotes[len(quotes)-1] == r:
				quotes = quotes[:len(quotes)-1]
				if len(quotes) == 0 {
					flush()
				} else {
					buf.WriteRune(r)
				}
			case len(quotes) == 0:
				quotes = append(quotes, r)
			default:
				buf.WriteRune(r)
			}
		case ' ':
			flush()
		default:
			buf.WriteRune(r)
		}
	}

	if len(quotes) > 0 {
		err = ErrUnterminatedQuotes
	} else {
		flush()
	}
	return
}
