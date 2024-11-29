package middleware

import (
	"net"
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
		Average int `validate:"min=1,required"`
		Burst   int `validate:"min=1,required"`
		Period  time.Duration
	}
)

var (
	RateLimiter            = &Middleware{withOptions: NewRateLimiter}
	rateLimiterOptsDefault = rateLimiterOpts{
		Period: time.Second,
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

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		rl.m.Debug().Msgf("unable to parse remote address %s", r.RemoteAddr)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	limiter, ok := rl.requestMap[host]
	if !ok {
		limiter = rl.newLimiter()
		rl.requestMap[host] = limiter
	}

	rl.mu.Unlock()

	if limiter.Allow() {
		next(w, r)
		return
	}

	http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
}
