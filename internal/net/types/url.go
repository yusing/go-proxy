package types

import (
	urlPkg "net/url"
)

type URL struct {
	*urlPkg.URL
}

func ParseURL(url string) (URL, error) {
	u, err := urlPkg.Parse(url)
	if err != nil {
		return URL{}, err
	}
	return URL{URL: u}, nil
}

func NewURL(url *urlPkg.URL) URL {
	return URL{url}
}

func (u URL) String() string {
	if u.URL == nil {
		return "nil"
	}
	return u.URL.String()
}

func (u URL) MarshalJSON() (text []byte, err error) {
	if u.URL == nil {
		return []byte("null"), nil
	}
	return []byte("\"" + u.URL.String() + "\""), nil
}

func (u URL) Equals(other *URL) bool {
	return u.URL == other.URL || u.String() == other.String()
}

func (u URL) JoinPath(path string) URL {
	return URL{u.URL.JoinPath(path)}
}
