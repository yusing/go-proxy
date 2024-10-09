package types

import "net/url"

type URL struct{ *url.URL }

func NewURL(url *url.URL) URL {
	return URL{url}
}

func (u URL) String() string {
	if u.URL == nil {
		return "nil"
	}
	return u.URL.String()
}

func (u URL) MarshalText() (text []byte, err error) {
	return []byte(u.String()), nil
}

func (u URL) Equals(other URL) bool {
	return u.URL == other.URL || u.String() == other.String()
}
