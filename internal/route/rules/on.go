package rules

import (
	"net"
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	RuleOn struct {
		raw   string
		check CheckFulfill
	}
	CheckFulfill func(r *http.Request) bool
	Checkers     []CheckFulfill
)

const (
	OnHeader = "header"
	OnQuery  = "query"
	OnMethod = "method"
	OnPath   = "path"
	OnRemote = "remote"
)

var checkers = map[string]struct {
	validate ValidateFunc
	check    func(r *http.Request, args any) bool
}{
	OnHeader: { // header <key> <value>
		validate: toStrTuple,
		check: func(r *http.Request, args any) bool {
			return r.Header.Get(args.(StrTuple).First) == args.(StrTuple).Second
		},
	},
	OnQuery: { // query <key> <value>
		validate: toStrTuple,
		check: func(r *http.Request, args any) bool {
			return r.URL.Query().Get(args.(StrTuple).First) == args.(StrTuple).Second
		},
	},
	OnMethod: { // method <method>
		validate: validateMethod,
		check: func(r *http.Request, method any) bool {
			return r.Method == method.(string)
		},
	},
	OnPath: { // path <path>
		validate: validateURLPath,
		check: func(r *http.Request, globPath any) bool {
			reqPath := r.URL.Path
			if len(reqPath) > 0 && reqPath[0] != '/' {
				reqPath = "/" + reqPath
			}
			return strutils.GlobMatch(globPath.(string), reqPath)
		},
	},
	OnRemote: { // remote <ip|cidr>
		validate: validateCIDR,
		check: func(r *http.Request, cidr any) bool {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return false
			}
			return cidr.(*net.IPNet).Contains(ip)
		},
	},
}

// Parse implements strutils.Parser.
func (on *RuleOn) Parse(v string) error {
	on.raw = v

	lines := strutils.SplitLine(v)
	checks := make(Checkers, 0, len(lines))

	errs := E.NewBuilder("rule.on syntax errors")
	for i, line := range lines {
		if line == "" {
			continue
		}
		parsed, err := parseOn(line)
		if err != nil {
			errs.Add(err.Subjectf("line %d", i+1))
			continue
		}
		checks = append(checks, parsed.matchOne())
	}

	on.check = checks.matchAll()
	return errs.Error()
}

func (on *RuleOn) String() string {
	return on.raw
}

func (on *RuleOn) MarshalJSON() ([]byte, error) {
	return []byte("\"" + on.String() + "\""), nil
}

func parseOn(line string) (Checkers, E.Error) {
	ors := strutils.SplitRune(line, '|')

	if len(ors) > 1 {
		errs := E.NewBuilder("rule.on syntax errors")
		checks := make([]CheckFulfill, len(ors))
		for i, or := range ors {
			curCheckers, err := parseOn(or)
			if err != nil {
				errs.Add(err)
				continue
			}
			checks[i] = curCheckers[0]
		}
		if err := errs.Error(); err != nil {
			return nil, err
		}
		return checks, nil
	}

	subject, args, err := parse(line)
	if err != nil {
		return nil, err
	}

	checker, ok := checkers[subject]
	if !ok {
		return nil, ErrInvalidOnTarget.Subject(subject)
	}

	validArgs, err := checker.validate(args)
	if err != nil {
		return nil, err.Subject(subject)
	}

	return Checkers{
		func(r *http.Request) bool {
			return checker.check(r, validArgs)
		},
	}, nil
}

func (checkers Checkers) matchOne() CheckFulfill {
	return func(r *http.Request) bool {
		for _, checker := range checkers {
			if checker(r) {
				return true
			}
		}
		return false
	}
}

func (checkers Checkers) matchAll() CheckFulfill {
	return func(r *http.Request) bool {
		for _, checker := range checkers {
			if !checker(r) {
				return false
			}
		}
		return true
	}
}
