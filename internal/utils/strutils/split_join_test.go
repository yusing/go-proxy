package strutils_test

import (
	"strings"
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/strutils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

var alphaNumeric = func() string {
	var s strings.Builder
	for i := range 'z' - 'a' + 1 {
		s.WriteRune('a' + i)
		s.WriteRune('A' + i)
		s.WriteRune(',')
	}
	for i := range '9' - '0' + 1 {
		s.WriteRune('0' + i)
		s.WriteRune(',')
	}
	return s.String()
}()

func TestSplit(t *testing.T) {
	tests := map[string]rune{
		"":  0,
		"1": '1',
		",": ',',
	}
	for sep, rsep := range tests {
		t.Run(sep, func(t *testing.T) {
			expected := strings.Split(alphaNumeric, sep)
			ExpectDeepEqual(t, SplitRune(alphaNumeric, rsep), expected)
			ExpectEqual(t, JoinRune(expected, rsep), alphaNumeric)
		})
	}
}
