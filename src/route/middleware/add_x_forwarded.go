package middleware

var AddXForwarded = &Middleware{
	rewrite: func(r *ProxyRequest) {
		r.SetXForwarded()
	},
}
