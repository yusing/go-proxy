package middleware

type (
	rateLimiter struct {
		*rateLimiterOpts
		m *Middleware
	}

	rateLimiterOpts struct {
		Count int `json:"count"`
	}
)
