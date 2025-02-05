package rules

import (
	"testing"

	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestParseOn(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr E.Error
	}{
		// header
		{
			name:    "header_valid_kv",
			input:   "header Connection Upgrade",
			wantErr: nil,
		},
		{
			name:    "header_valid_k",
			input:   "header Connection",
			wantErr: nil,
		},
		{
			name:    "header_missing_arg",
			input:   "header",
			wantErr: ErrExpectKVOptionalV,
		},
		// query
		{
			name:    "query_valid_kv",
			input:   "query key value",
			wantErr: nil,
		},
		{
			name:    "query_valid_k",
			input:   "query key",
			wantErr: nil,
		},
		{
			name:    "query_missing_arg",
			input:   "query",
			wantErr: ErrExpectKVOptionalV,
		},
		{
			name:    "cookie_valid_kv",
			input:   "cookie key value",
			wantErr: nil,
		},
		{
			name:    "cookie_valid_k",
			input:   "cookie key",
			wantErr: nil,
		},
		{
			name:    "cookie_missing_arg",
			input:   "cookie",
			wantErr: ErrExpectKVOptionalV,
		},
		// method
		{
			name:    "method_valid",
			input:   "method GET",
			wantErr: nil,
		},
		{
			name:    "method_invalid",
			input:   "method invalid",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "method_missing_arg",
			input:   "method",
			wantErr: ErrExpectOneArg,
		},
		// path
		{
			name:    "path_valid",
			input:   "path /home",
			wantErr: nil,
		},
		{
			name:    "path_missing_arg",
			input:   "path",
			wantErr: ErrExpectOneArg,
		},
		// remote
		{
			name:    "remote_valid",
			input:   "remote 127.0.0.1",
			wantErr: nil,
		},
		{
			name:    "remote_invalid",
			input:   "remote abcd",
			wantErr: ErrInvalidArguments,
		},
		{
			name:    "remote_missing_arg",
			input:   "remote",
			wantErr: ErrExpectOneArg,
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
