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
			name:  "relative2",
			input: "/icon.png",
			wantValue: &IconURL{
				Value:      "/icon.png",
				IconSource: IconSourceRelative,
			},
		},
		{
			name:    "relative_empty_path",
			input:   "@target/",
			wantErr: true,
		},
		{
			name:    "relative_empty_path2",
			input:   "/",
			wantErr: true,
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
			name:    "walkxcode_invalid_format",
			input:   "foo/walkxcode.png",
			wantErr: true,
		},
		{
			name:  "selfh.st_valid",
			input: "@selfhst/foo.png",
			wantValue: &IconURL{
				Value:      "/foo.png",
				IconSource: IconSourceSelfhSt,
				Extra: &IconExtra{
					FileType: "png",
					Name:     "foo",
				},
			},
		},
		{
			name:    "selfh.st_invalid",
			input:   "@selfhst/foo",
			wantErr: true,
		},
		{
			name:    "selfh.st_invalid_format",
			input:   "@selfhst/foo.bar",
			wantErr: true,
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
