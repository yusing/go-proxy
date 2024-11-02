package middleware

import (
	"net/http"
	"sync"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"golang.org/x/time/rate"
)

type (
	requestMap  = map[string]*rate.Limiter
	rateLimiter struct {
		requestMap requestMap
		newLimiter func() *rate.Limiter
		m          *Middleware

		mu sync.Mutex
	}

	rateLimiterOpts struct {
		Average int           `json:"average"`
		Burst   int           `json:"burst"`
		Period  time.Duration `json:"period"`
	}
)

var (
	RateLimiter            = &Middleware{withOptions: NewRateLimiter}
	rateLimiterOptsDefault = rateLimiterOpts{
		Average: 100,
		Burst:   1,
		Period:  time.Second,
	}
)

func NewRateLimiter(optsRaw OptionsRaw) (*Middleware, E.Error) {
	rl := new(rateLimiter)
	opts := rateLimiterOptsDefault
	err := Deserialize(optsRaw, &opts)
	if err != nil {
		return nil, err
	}
	switch {
	case opts.Average == 0:
		return nil, ErrZeroValue.Subject("average")
	case opts.Burst == 0:
		return nil, ErrZeroValue.Subject("burst")
	case opts.Period == 0:
		return nil, ErrZeroValue.Subject("period")
	}
	rl.requestMap = make(requestMap, 0)
	rl.newLimiter = func() *rate.Limiter {
		return rate.NewLimiter(rate.Limit(opts.Average)*rate.Every(opts.Period), opts.Burst)
	}
	rl.m = &Middleware{
		impl:   rl,
		before: rl.limit,
	}
	return rl.m, nil
}

func (rl *rateLimiter) limit(next http.HandlerFunc, w ResponseWriter, r *Request) {
	rl.mu.Lock()

	limiter, ok := rl.requestMap[r.RemoteAddr]
	if !ok {
		limiter = rl.newLimiter()
		rl.requestMap[r.RemoteAddr] = limiter
	}

	rl.mu.Unlock()

	if limiter.Allow() {
		next(w, r)
		return
	}

	http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
}
