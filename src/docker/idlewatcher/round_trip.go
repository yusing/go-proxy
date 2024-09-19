package idlewatcher

import "net/http"

type (
	roundTripper struct {
		patched roundTripFunc
	}
	roundTripFunc func(*http.Request) (*http.Response, error)
)

func (rt roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.patched(req)
}
