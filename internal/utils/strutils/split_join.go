package strutils

import (
	"math"
	"strings"
)

// SplitRune is like strings.Split but takes a rune as separator.
func SplitRune(s string, sep rune) []string {
	if sep == 0 {
		return strings.Split(s, "")
	}
	n := strings.Count(s, string(sep)) + 1
	if n > len(s)+1 {
		n = len(s) + 1
	}
	a := make([]string, n)
	n--
	i := 0
	for i < n {
		m := strings.IndexRune(s, sep)
		if m < 0 {
			break
		}
		a[i] = s[:m]
		s = s[m+1:]
		i++
	}
	a[i] = s
	return a[:i+1]
}

// SplitComma is a wrapper around SplitRune(s, ',').
func SplitComma(s string) []string {
	return SplitRune(s, ',')
}

// SplitLine is a wrapper around SplitRune(s, '\n').
func SplitLine(s string) []string {
	return SplitRune(s, '\n')
}

// SplitSpace is a wrapper around SplitRune(s, ' ').
func SplitSpace(s string) []string {
	return SplitRune(s, ' ')
}

// JoinRune is like strings.Join but takes a rune as separator.
func JoinRune(elems []string, sep rune) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	if sep == 0 {
		return strings.Join(elems, "")
	}

	var n int
	for _, elem := range elems {
		if len(elem) > math.MaxInt-n {
			panic("strings: Join output length overflow")
		}
		n += len(elem)
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		b.WriteRune(sep)
		b.WriteString(s)
	}
	return b.String()
}

// JoinLines is a wrapper around JoinRune(elems, '\n').
func JoinLines(elems []string) string {
	return JoinRune(elems, '\n')
}
