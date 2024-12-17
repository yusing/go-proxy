package accesslog

import (
	"strconv"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type StatusCodeRange struct {
	Start int
	End   int
}

var ErrInvalidStatusCodeRange = E.New("invalid status code range")

func (r *StatusCodeRange) Includes(code int) bool {
	return r.Start <= code && code <= r.End
}

func (r *StatusCodeRange) Parse(v string) error {
	split := strings.Split(v, "-")
	switch len(split) {
	case 1:
		start, err := strconv.Atoi(split[0])
		if err != nil {
			return E.From(err)
		}
		r.Start = start
		r.End = start
		return nil
	case 2:
		start, errStart := strconv.Atoi(split[0])
		end, errEnd := strconv.Atoi(split[1])
		if err := E.Join(errStart, errEnd); err != nil {
			return err
		}
		r.Start = start
		r.End = end
		return nil
	default:
		return ErrInvalidStatusCodeRange.Subject(v)
	}
}

func (r *StatusCodeRange) String() string {
	if r.Start == r.End {
		return strconv.Itoa(r.Start)
	}
	return strconv.Itoa(r.Start) + "-" + strconv.Itoa(r.End)
}
