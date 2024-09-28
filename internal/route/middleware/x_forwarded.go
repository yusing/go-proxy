package middleware

var AddXForwarded = &Middleware{
	rewrite: (*ProxyRequest).AddXForwarded,
}

var SetXForwarded = &Middleware{
	rewrite: (*ProxyRequest).SetXForwarded,
}
