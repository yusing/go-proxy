package rules

import (
	"testing"

	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestParseSubjectArgs(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		subject, args, err := parse("rewrite / /foo/bar")
		ExpectNoError(t, err)
		ExpectEqual(t, subject, "rewrite")
		ExpectDeepEqual(t, args, []string{"/", "/foo/bar"})
	})
	t.Run("with quotes", func(t *testing.T) {
		subject, args, err := parse(`error 403 "Forbidden 'foo' 'bar'."`)
		ExpectNoError(t, err)
		ExpectEqual(t, subject, "error")
		ExpectDeepEqual(t, args, []string{"403", "Forbidden 'foo' 'bar'."})
	})
	t.Run("with escaped", func(t *testing.T) {
		subject, args, err := parse(`error 403 Forbidden\ \"foo\"\ \"bar\".`)
		ExpectNoError(t, err)
		ExpectEqual(t, subject, "error")
		ExpectDeepEqual(t, args, []string{"403", "Forbidden \"foo\" \"bar\"."})
	})
}

func TestParseCommands(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// bypass tests
		{
			name:    "bypass_valid",
			input:   "bypass",
			wantErr: nil,
		},
		{
			name:    "bypass_invalid_with_args",
			input:   "bypass /",
			wantErr: ErrInvalidArguments,
		},
		// rewrite tests
		{
			name:    "rewrite_valid",
			input:   "rewrite / /foo/bar",
			wantErr: nil,
		},
		{
			name:    "rewrite_missing_target",
			input:   "rewrite /",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "rewrite_too_many_args",
			input:   "rewrite / / /",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "rewrite_no_leading_slash",
			input:   "rewrite abc /",
			wantErr: ErrInvalidArguments,
		},
		// serve tests
		{
			name:    "serve_valid",
			input:   "serve /var/www",
			wantErr: nil,
		},
		{
			name:    "serve_missing_path",
			input:   "serve ",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "serve_too_many_args",
			input:   "serve / / /",
			wantErr: ErrInvalidArguments,
		},
		// redirect tests
		{
			name:    "redirect_valid",
			input:   "redirect /",
			wantErr: nil,
		},
		{
			name:    "redirect_too_many_args",
			input:   "redirect / /",
			wantErr: ErrInvalidArguments,
		},
		// error directive tests
		{
			name:    "error_valid",
			input:   "error 404 Not\\ Found",
			wantErr: nil,
		},
		{
			name:    "error_missing_status_code",
			input:   "error Not\\ Found",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "error_too_many_args",
			input:   "error 404 Not\\ Found extra",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "error_no_escaped_space",
			input:   "error 404 Not Found",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "error_invalid_status_code",
			input:   "error 123 abc",
			wantErr: ErrInvalidArguments,
		},
		// proxy directive tests
		{
			name:    "proxy_valid",
			input:   "proxy localhost:8080",
			wantErr: nil,
		},
		{
			name:    "proxy_missing_target",
			input:   "proxy",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "proxy_too_many_args",
			input:   "proxy localhost:8080 extra",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "proxy_invalid_url",
			input:   "proxy :invalid_url",
			wantErr: ErrInvalidArguments,
		},
		// unknown directive test
		{
			name:    "unknown_directive",
			input:   "unknown /",
			wantErr: ErrUnknownDirective,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command{}
			err := cmd.Parse(tt.input)
			if tt.wantErr != nil {
				ExpectError(t, tt.wantErr, err)
			} else {
				ExpectNoError(t, err)
			}
		})
	}
}

func TestParseOn(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr E.Error
	}{
		// header
		{
			name:    "header_valid",
			input:   "header Connection Upgrade",
			wantErr: nil,
		},
		{
			name:    "header_invalid",
			input:   "header Connection",
			wantErr: ErrInvalidArguments,
		},
		// query
		{
			name:    "query_valid",
			input:   "query key value",
			wantErr: nil,
		},
		{
			name:    "query_invalid",
			input:   "query key",
			wantErr: ErrInvalidArguments,
		},
		// method
		{
			name:    "method_valid",
			input:   "method GET",
			wantErr: nil,
		},
		{
			name:    "method_invalid",
			input:   "method",
			wantErr: ErrInvalidArguments,
		},
		// path
		{
			name:    "path_valid",
			input:   "path /home",
			wantErr: nil,
		},
		{
			name:    "path_invalid",
			input:   "path",
			wantErr: ErrInvalidArguments,
		},
		// remote
		{
			name:    "remote_valid",
			input:   "remote 127.0.0.1",
			wantErr: nil,
		},
		{
			name:    "remote_invalid",
			input:   "remote",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "unknown_target",
			input:   "unknown",
			wantErr: ErrInvalidOnTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			on := &RuleOn{}
			err := on.Parse(tt.input)
			if tt.wantErr != nil {
				ExpectError(t, tt.wantErr, err)
			} else {
				ExpectNoError(t, err)
			}
		})
	}
}

func TestParseRule(t *testing.T) {
	// test := map[string]any{
	// 	"name": "test",
	// 	"on":   "method GET",
	// 	"do":   "bypass",
	// }
}
