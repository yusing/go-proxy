package types

import (
	"net/http"
	"path"
	"strconv"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	Rules []Rule
	Rule  struct {
		Name string  `json:"name" validate:"required,unique"`
		On   RuleOn  `json:"on"`
		Do   Command `json:"do"`
	}
	RuleOn struct {
		raw      string
		checkers []CheckFulfill
	}
	Command struct {
		raw string
		CommandExecutor
	}
	CheckFulfill           func(r *http.Request) bool
	RequestObjectRetriever struct {
		expectedArgs int
		retrieve     func(r *http.Request, args []string) string
		equal        func(v, want string) bool
	}
	CommandExecutor struct {
		http.HandlerFunc
		proceed bool
	}
	CommandBuilder struct {
		expectedArgs int
		build        func(args []string) CommandExecutor
	}
)

/*
proxy.app1.rules: |
	- name: default
		do: |
			rewrite / /index.html
			serve /var/www/goaccess
	- name: ws
		on: |
			header Connection upgrade
			header Upgrade websocket
		do: proxy $upstream_url
*/

var (
	ErrUnterminatedQuotes    = E.New("unterminated quotes")
	ErrUnsupportedEscapeChar = E.New("unsupported escape char")
	ErrUnknownDirective      = E.New("unknown directive")
	ErrInvalidArguments      = E.New("invalid arguments")
	ErrInvalidCriteria       = E.New("invalid criteria")
	ErrInvalidCriteriaTarget = E.New("invalid criteria target")
)

var retrievers = map[string]RequestObjectRetriever{
	"header": {1, func(r *http.Request, args []string) string {
		return r.Header.Get(args[0])
	}, nil},
	"query": {1, func(r *http.Request, args []string) string {
		return r.URL.Query().Get(args[0])
	}, nil},
	"method": {0, func(r *http.Request, _ []string) string {
		return r.Method
	}, nil},
	"path": {0, func(r *http.Request, _ []string) string {
		return r.URL.Path
	}, func(v, want string) bool {
		return strutils.GlobMatch(want, v)
	}},
	"remote": {0, func(r *http.Request, _ []string) string {
		return r.RemoteAddr
	}, nil},
}

var commands = map[string]CommandBuilder{
	"rewrite": {2, func(args []string) CommandExecutor {
		orig, repl := args[0], args[1]
		return CommandExecutor{
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				r.URL.Path = strings.Replace(r.URL.Path, orig, repl, 1)
				r.URL.RawPath = r.URL.EscapedPath()
				r.RequestURI = r.URL.String()
			},
			proceed: true,
		}
	}},
	"serve": {1, func(args []string) CommandExecutor {
		return CommandExecutor{
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, path.Join(args[0], path.Clean(r.URL.Path)))
			},
			proceed: false,
		}
	}},
	"redirect": {1, func(args []string) CommandExecutor {
		target := args[0]
		return CommandExecutor{
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			},
			proceed: false,
		}
	}},
	"error": {2, func(args []string) CommandExecutor {
		codeStr, text := args[0], args[1]
		code, err := strconv.Atoi(codeStr)
		if err != nil {
			code = http.StatusNotFound
		}
		return CommandExecutor{
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, text, code)
			},
			proceed: false,
		}
	}},
	"proxy": {1, func(args []string) CommandExecutor {
		target := args[0]
		return CommandExecutor{
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = target
				r.URL.RawPath = r.URL.EscapedPath()
				r.RequestURI = r.URL.String()
			},
			proceed: true,
		}
	}},
}

var escapedChars = map[rune]rune{
	'n':  '\n',
	't':  '\t',
	'r':  '\r',
	'\'': '\'',
	'"':  '"',
	' ':  ' ',
}

// BuildHandler returns a http.HandlerFunc that implements the rules.
//
//	Bypass rules are executed first
//	if a bypass rule matches,
//	the request is passed to the upstream and no more rules are executed.
//
//	Other rules are executed later
//	if no rule matches, the default rule is executed
//	if no rule matches and default rule is not set,
//	the request is passed to the upstream.
func (rules Rules) BuildHandler(up *gphttp.ReverseProxy) http.HandlerFunc {
	// move bypass rules to the front.
	bypassRules := make(Rules, 0, len(rules))
	otherRules := make(Rules, 0, len(rules))

	var defaultRule Rule

	for _, rule := range rules {
		switch {
		case rule.Do.isBypass():
			bypassRules = append(bypassRules, rule)
		case rule.Name == "default":
			defaultRule = rule
		default:
			otherRules = append(otherRules, rule)
		}
	}

	// free allocated empty slices
	// before passing them to the handler.
	if len(bypassRules) == 0 {
		bypassRules = []Rule{}
	}
	if len(otherRules) == 0 {
		otherRules = []Rule{defaultRule}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		hasMatch := false
		for _, rule := range bypassRules {
			if rule.On.MatchAll(r) {
				up.ServeHTTP(w, r)
				return
			}
		}
		for _, rule := range otherRules {
			if rule.On.MatchAll(r) {
				hasMatch = true
				rule.Do.HandlerFunc(w, r)
				if !rule.Do.proceed {
					return
				}
			}
		}
		if hasMatch || defaultRule.Do.isBypass() {
			up.ServeHTTP(w, r)
			return
		}

		defaultRule.Do.HandlerFunc(w, r)
		if !defaultRule.Do.proceed {
			return
		}
	}
}

// parse line to subject and args
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

func (on *RuleOn) Parse(v string) E.Error {
	lines := strutils.SplitLine(v)
	on.checkers = make([]CheckFulfill, 0, len(lines))
	on.raw = v

	errs := E.NewBuilder("rule.on syntax errors")
	for i, line := range lines {
		subject, args, err := parse(line)
		if err != nil {
			errs.Add(err.Subjectf("line %d", i+1))
			continue
		}
		retriever, ok := retrievers[subject]
		if !ok {
			errs.Add(ErrInvalidCriteriaTarget.Subject(subject).Subjectf("line %d", i+1))
			continue
		}
		nArgs := retriever.expectedArgs
		if len(args) != nArgs+1 {
			errs.Add(ErrInvalidArguments.Subject(subject).Subjectf("line %d", i+1))
			continue
		}
		equal := retriever.equal
		if equal == nil {
			equal = func(a, b string) bool {
				return a == b
			}
		}
		on.checkers = append(on.checkers, func(r *http.Request) bool {
			return equal(retriever.retrieve(r, args[:nArgs]), args[nArgs])
		})
	}
	return errs.Error()
}

func (on *RuleOn) MatchAll(r *http.Request) bool {
	for _, match := range on.checkers {
		if !match(r) {
			return false
		}
	}
	return true
}

func (cmd *Command) Parse(v string) E.Error {
	cmd.raw = v
	directive, args, err := parse(v)
	if err != nil {
		return err
	}

	if directive == "bypass" {
		if len(args) != 0 {
			return ErrInvalidArguments.Subject(directive)
		}
		return nil
	}

	builder, ok := commands[directive]
	if !ok {
		return ErrUnknownDirective.Subject(directive)
	}
	if len(args) != builder.expectedArgs {
		return ErrInvalidArguments.Subject(directive)
	}
	cmd.CommandExecutor = builder.build(args)
	return nil
}

func (cmd *Command) isBypass() bool {
	return cmd.HandlerFunc == nil
}

func (on *RuleOn) String() string {
	return on.raw
}

func (on *RuleOn) MarshalJSON() ([]byte, error) {
	return []byte("\"" + on.String() + "\""), nil
}

func (cmd *Command) String() string {
	return cmd.raw
}

func (cmd *Command) MarshalJSON() ([]byte, error) {
	return []byte("\"" + cmd.String() + "\""), nil
}
