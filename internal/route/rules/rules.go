package rules

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
)

type (
	/*
		Example:

			proxy.app1.rules: |
				- name: default
					do: |
						rewrite / /index.html
						serve /var/www/goaccess
				- name: ws
					on: |
						header Connection Upgrade
						header Upgrade websocket
					do: bypass

			proxy.app2.rules: |
				- name: default
					do: bypass
				- name: block POST and PUT
					on: method POST | method PUT
					do: error 403 Forbidden
	*/
	Rules []Rule
	/*
		Rule is a rule for a reverse proxy.
		It do `Do` when `On` matches.

		A rule can have multiple lines of on.

		All lines of on must match,
		but each line can have multiple checks that
		one match means this line is matched.
	*/
	Rule struct {
		Name string  `json:"name" validate:"required,unique"`
		On   RuleOn  `json:"on"`
		Do   Command `json:"do"`
	}
)

// BuildHandler returns a http.HandlerFunc that implements the rules.
//
//	Bypass rules are executed first
//	if a bypass rule matches,
//	the request is passed to the upstream and no more rules are executed.
//
//	Other rules are executed later
//	if no rule matches, the default rule is executed
//	if no rule matches and default rule is not set,
//	the request is passed to the upstream.
func (rules Rules) BuildHandler(up *reverseproxy.ReverseProxy) http.HandlerFunc {
	// move bypass rules to the front.
	bypassRules := make(Rules, 0, len(rules))
	otherRules := make(Rules, 0, len(rules))

	var defaultRule Rule

	for _, rule := range rules {
		switch {
		case rule.Do.isBypass():
			bypassRules = append(bypassRules, rule)
		case rule.Name == "default":
			defaultRule = rule
		default:
			otherRules = append(otherRules, rule)
		}
	}

	// free allocated empty slices
	// before encapsulating them into the handlerFunc.
	if len(bypassRules) == 0 {
		bypassRules = []Rule{}
	}
	if len(otherRules) == 0 {
		otherRules = []Rule{defaultRule}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		for _, rule := range bypassRules {
			if rule.On.check(r) {
				up.ServeHTTP(w, r)
				return
			}
		}
		hasMatch := false
		for _, rule := range otherRules {
			if rule.On.check(r) {
				hasMatch = true
				rule.Do.HandlerFunc(w, r)
				if !rule.Do.proceed {
					return
				}
			}
		}
		if hasMatch || defaultRule.Do.isBypass() {
			up.ServeHTTP(w, r)
			return
		}

		defaultRule.Do.HandlerFunc(w, r)
	}
}
