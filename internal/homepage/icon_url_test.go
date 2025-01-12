package homepage

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestIconURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue *IconURL
		wantErr   bool
	}{
		{
			name:  "absolute",
			input: "http://example.com/icon.png",
			wantValue: &IconURL{
				Value:      "http://example.com/icon.png",
				IconSource: IconSourceAbsolute,
			},
		},
		{
			name:  "relative",
			input: "@target/icon.png",
			wantValue: &IconURL{
				Value:      "/icon.png",
				IconSource: IconSourceRelative,
			},
		},
		{
			name:  "walkxcode",
			input: "png/walkxcode.png",
			wantValue: &IconURL{
				Value:      "png/walkxcode.png",
				IconSource: IconSourceWalkXCode,
				Extra: &IconExtra{
					FileType: "png",
					Name:     "walkxcode",
				},
			},
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := &IconURL{}
			err := u.Parse(tc.input)
			if tc.wantErr {
				ExpectError(t, ErrInvalidIconURL, err)
			} else {
				ExpectNoError(t, err)
				ExpectDeepEqual(t, u, tc.wantValue)
			}
		})
	}
}
