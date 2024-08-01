package functional

import (
	"fmt"
	"strconv"
	"strings"
)

type Stringable struct{ string }

func NewStringable(v any) Stringable {
	switch vv := v.(type) {
	case string:
		return Stringable{vv}
	case fmt.Stringer:
		return Stringable{vv.String()}
	default:
		return Stringable{fmt.Sprint(vv)}
	}
}

func (s Stringable) String() string {
	return s.string
}

func (s Stringable) Len() int {
	return len(s.string)
}

func (s Stringable) MarshalText() (text []byte, err error) {
	return []byte(s.string), nil
}

func (s Stringable) SubStr(start int, end int) Stringable {
	return Stringable{s.string[start:end]}
}

func (s Stringable) HasPrefix(p Stringable) bool {
	return len(s.string) >= len(p.string) && s.string[0:len(p.string)] == p.string
}

func (s Stringable) HasSuffix(p Stringable) bool {
	return len(s.string) >= len(p.string) && s.string[len(s.string)-len(p.string):] == p.string
}

func (s Stringable) IsEmpty() bool {
	return len(s.string) == 0
}

func (s Stringable) IndexRune(r rune) int {
	return strings.IndexRune(s.string, r)
}

func (s Stringable) ToInt() (int, error) {
	return strconv.Atoi(s.string)
}

func (s Stringable) Split(sep string) []Stringable {
	return Stringables(strings.Split(s.string, sep))
}

func Stringables(ss []string) []Stringable {
	ret := make([]Stringable, len(ss))
	for i, s := range ss {
		ret[i] = Stringable{s}
	}
	return ret
}
