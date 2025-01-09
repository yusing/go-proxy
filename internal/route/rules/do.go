package rules

import (
	"net/http"
	"path"
	"strconv"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	Command struct {
		raw  string
		exec *CommandExecutor
	}
	CommandExecutor struct {
		directive string
		http.HandlerFunc
		proceed bool
	}
)

const (
	CommandRewrite  = "rewrite"
	CommandServe    = "serve"
	CommandProxy    = "proxy"
	CommandRedirect = "redirect"
	CommandError    = "error"
	CommandBypass   = "bypass"
)

var commands = map[string]struct {
	help     Help
	validate ValidateFunc
	build    func(args any) *CommandExecutor
}{
	CommandRewrite: {
		help: Help{
			command: CommandRewrite,
			args: map[string]string{
				"from": "the path to rewrite, must start with /",
				"to":   "the path to rewrite to, must start with /",
			},
		},
		validate: func(args []string) (any, E.Error) {
			if len(args) != 2 {
				return nil, ErrExpectTwoArgs
			}
			return validateURLPaths(args)
		},
		build: func(args any) *CommandExecutor {
			a := args.([]string)
			orig, repl := a[0], a[1]
			return &CommandExecutor{
				HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
					path := r.URL.Path
					if len(path) > 0 && path[0] != '/' {
						path = "/" + path
					}
					if !strings.HasPrefix(path, orig) {
						return
					}
					path = repl + path[len(orig):]
					r.URL.Path = path
					r.URL.RawPath = r.URL.EscapedPath()
					r.RequestURI = r.URL.RequestURI()
				},
				proceed: true,
			}
		},
	},
	CommandServe: {
		help: Help{
			command: CommandServe,
			args: map[string]string{
				"root": "the file system path to serve, must be an existing directory",
			},
		},
		validate: validateFSPath,
		build: func(args any) *CommandExecutor {
			root := args.(string)
			return &CommandExecutor{
				HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, path.Join(root, path.Clean(r.URL.Path)))
				},
				proceed: false,
			}
		},
	},
	CommandRedirect: {
		help: Help{
			command: CommandRedirect,
			args: map[string]string{
				"to": "the url to redirect to, can be relative or absolute URL",
			},
		},
		validate: validateURL,
		build: func(args any) *CommandExecutor {
			target := args.(types.URL).String()
			return &CommandExecutor{
				HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, target, http.StatusTemporaryRedirect)
				},
				proceed: false,
			}
		},
	},
	CommandError: {
		help: Help{
			command: CommandError,
			args: map[string]string{
				"code": "the http status code to return",
				"text": "the error message to return",
			},
		},
		validate: func(args []string) (any, E.Error) {
			if len(args) != 2 {
				return nil, ErrExpectTwoArgs
			}
			codeStr, text := args[0], args[1]
			code, err := strconv.Atoi(codeStr)
			if err != nil {
				return nil, ErrInvalidArguments.With(err)
			}
			if !gphttp.IsStatusCodeValid(code) {
				return nil, ErrInvalidArguments.Subject(codeStr)
			}
			return []any{code, text}, nil
		},
		build: func(args any) *CommandExecutor {
			a := args.([]any)
			code, text := a[0].(int), a[1].(string)
			return &CommandExecutor{
				HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, text, code)
				},
				proceed: false,
			}
		},
	},
	CommandProxy: {
		help: Help{
			command: CommandProxy,
			args: map[string]string{
				"to": "the url to proxy to, must be an absolute URL",
			},
		},
		validate: validateAbsoluteURL,
		build: func(args any) *CommandExecutor {
			target := args.(types.URL)
			if target.Scheme == "" {
				target.Scheme = "http"
			}
			rp := reverseproxy.NewReverseProxy("", target, gphttp.DefaultTransport)
			return &CommandExecutor{
				HandlerFunc: rp.ServeHTTP,
				proceed:     false,
			}
		},
	},
}

// Parse implements strutils.Parser.
func (cmd *Command) Parse(v string) error {
	cmd.raw = v

	lines := strutils.SplitLine(v)
	if len(lines) == 0 {
		return nil
	}

	executors := make([]*CommandExecutor, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		directive, args, err := parse(line)
		if err != nil {
			return err
		}

		if directive == CommandBypass {
			if len(args) != 0 {
				return ErrInvalidArguments.Subject(directive)
			}
			return nil
		}

		builder, ok := commands[directive]
		if !ok {
			return ErrUnknownDirective.Subject(directive)
		}
		validArgs, err := builder.validate(args)
		if err != nil {
			return err.Subject(directive).Withf("%s", builder.help.String())
		}

		exec := builder.build(validArgs)
		exec.directive = directive
		executors = append(executors, exec)
	}

	exec, err := buildCmd(executors)
	if err != nil {
		return err
	}
	cmd.exec = exec
	return nil
}

func buildCmd(executors []*CommandExecutor) (*CommandExecutor, error) {
	for i, exec := range executors {
		if !exec.proceed && i != len(executors)-1 {
			return nil, ErrInvalidCommandSequence.
				Withf("%s cannot follow %s", exec, executors[i+1])
		}
	}
	return &CommandExecutor{
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			for _, exec := range executors {
				exec.HandlerFunc(w, r)
			}
		},
		proceed: executors[len(executors)-1].proceed,
	}, nil
}

func (cmd *Command) isBypass() bool {
	return cmd.exec == nil
}

func (cmd *Command) String() string {
	return cmd.raw
}

func (cmd *Command) MarshalJSON() ([]byte, error) {
	return []byte("\"" + cmd.String() + "\""), nil
}

func (exec *CommandExecutor) String() string {
	return exec.directive
}
