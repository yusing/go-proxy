package types

import (
	"testing"

	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/utils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestValidateConfig(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want gperr.Error
	}{
		{
			name: "valid config",
			data: []byte(`
autocert:
  provider: local
`),
			want: nil,
		},
		{
			name: "unknown field",
			data: []byte(`
autocert:
  provider: local
  unknown: true
`),
			want: utils.ErrUnknownField,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Validate(c.data)
			ExpectError(t, c.want, got)
		})
	}
}
