package middleware

var SetXForwarded = &Middleware{
	rewrite: func(r *ProxyRequest) {
		r.Out.Header.Del("X-Forwarded-For")
		r.Out.Header.Del("X-Forwarded-Host")
		r.Out.Header.Del("X-Forwarded-Proto")
		r.SetXForwarded()
	},
}
