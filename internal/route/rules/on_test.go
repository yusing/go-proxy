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
