package http

import "net/http"

type ServeMux struct {
	*http.ServeMux
}

func NewServeMux() ServeMux {
	return ServeMux{http.NewServeMux()}
}

func (mux ServeMux) HandleFunc(pattern string, handler http.HandlerFunc) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	mux.ServeMux.HandleFunc(pattern, handler)
	return
}
