package rules

import "net/http"

type (
	CheckFunc func(cached Cache, r *http.Request) bool
	Checker   interface {
		Check(cached Cache, r *http.Request) bool
	}
	CheckMatchSingle []Checker
	CheckMatchAll    []Checker
)

func (checker CheckFunc) Check(cached Cache, r *http.Request) bool {
	return checker(cached, r)
}

func (checkers CheckMatchSingle) Check(cached Cache, r *http.Request) bool {
	for _, check := range checkers {
		if check.Check(cached, r) {
			return true
		}
	}
	return false
}

func (checkers CheckMatchAll) Check(cached Cache, r *http.Request) bool {
	for _, check := range checkers {
		if !check.Check(cached, r) {
			return false
		}
	}
	return true
}
