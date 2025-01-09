package rules

import (
	"net/http"
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
		Name string  `json:"name" validate:"required"`
		On   RuleOn  `json:"on"`
		Do   Command `json:"do"`
	}
)

// BuildHandler returns a http.HandlerFunc that implements the rules.
//
//	if a bypass rule matches,
//	the request is passed to the upstream and no more rules are executed.
//
//	if no rule matches, the default rule is executed
//	if no rule matches and default rule is not set,
//	the request is passed to the upstream.
func (rules Rules) BuildHandler(up http.Handler) http.HandlerFunc {
	var (
		defaultRule      Rule
		defaultRuleIndex int
	)

	for i, rule := range rules {
		if rule.Name == "default" {
			defaultRule = rule
			defaultRuleIndex = i
			break
		}
	}

	rules = append(rules[:defaultRuleIndex], rules[defaultRuleIndex+1:]...)

	// free allocated empty slices
	// before encapsulating them into the handlerFunc.
	if len(rules) == 0 {
		if defaultRule.Do.isBypass() {
			return up.ServeHTTP
		}
		rules = []Rule{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		hasMatch := false
		for _, rule := range rules {
			if rule.On.check(r) {
				if rule.Do.isBypass() {
					up.ServeHTTP(w, r)
					return
				}
				rule.Do.exec.HandlerFunc(w, r)
				if !rule.Do.exec.proceed {
					return
				}
				hasMatch = true
			}
		}

		if hasMatch || defaultRule.Do.isBypass() {
			up.ServeHTTP(w, r)
			return
		}

		defaultRule.Do.exec.HandlerFunc(w, r)
	}
}
