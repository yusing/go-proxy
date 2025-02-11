package period

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type (
	PollFunc[T any]                  func(ctx context.Context) (*T, error)
	AggregateFunc[T, AggregateT any] func(entries ...*T) AggregateT
	Poller[T, AggregateT any]        struct {
		name       string
		poll       PollFunc[T]
		aggregator AggregateFunc[T, AggregateT]
		period     *Period[T]
		interval   time.Duration
		lastResult *T
		errs       []pollErr
	}
	pollErr struct {
		err   error
		count int
	}
)

const gatherErrsInterval = 30 * time.Second

func NewPoller[T any](
	name string,
	interval time.Duration,
	poll PollFunc[T],
) *Poller[T, T] {
	return &Poller[T, T]{
		name:     name,
		poll:     poll,
		period:   NewPeriod[T](),
		interval: interval,
	}
}

func NewPollerWithAggregator[T, AggregateT any](
	name string,
	interval time.Duration,
	poll PollFunc[T],
	aggregator AggregateFunc[T, AggregateT],
) *Poller[T, AggregateT] {
	return &Poller[T, AggregateT]{
		name:       name,
		poll:       poll,
		aggregator: aggregator,
		period:     NewPeriod[T](),
		interval:   interval,
	}
}

func (p *Poller[T, AggregateT]) appendErr(err error) {
	if len(p.errs) == 0 {
		p.errs = []pollErr{
			{err: err, count: 1},
		}
		return
	}
	for i, e := range p.errs {
		if e.err.Error() == err.Error() {
			p.errs[i].count++
			return
		}
	}
	p.errs = append(p.errs, pollErr{err: err, count: 1})
}

func (p *Poller[T, AggregateT]) gatherErrs() (string, bool) {
	if len(p.errs) == 0 {
		return "", false
	}
	title := fmt.Sprintf("Poller %s has encountered %d errors in the last %s seconds:", p.name, len(p.errs), gatherErrsInterval)
	errs := make([]string, 0, len(p.errs)+1)
	errs = append(errs, title)
	for _, e := range p.errs {
		errs = append(errs, fmt.Sprintf("%s: %d times", e.err.Error(), e.count))
	}
	return strings.Join(errs, "\n"), true
}

func (p *Poller[T, AggregateT]) pollWithTimeout(ctx context.Context) (*T, error) {
	ctx, cancel := context.WithTimeout(ctx, p.interval)
	defer cancel()
	return p.poll(ctx)
}

func (p *Poller[T, AggregateT]) Start() {
	go func() {
		ctx := task.RootContext()
		ticker := time.NewTicker(p.interval)
		gatherErrsTicker := time.NewTicker(gatherErrsInterval)
		defer ticker.Stop()
		defer gatherErrsTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := p.pollWithTimeout(ctx)
				if err != nil {
					p.appendErr(err)
					continue
				}
				p.period.Add(data)
				p.lastResult = data
			case <-gatherErrsTicker.C:
				errs, ok := p.gatherErrs()
				if ok {
					logging.Error().Msg(errs)
				}
			}
		}
	}()
}

func (p *Poller[T, AggregateT]) Get(filter Filter) []*T {
	return p.period.Get(filter)
}

func (p *Poller[T, AggregateT]) GetLastResult() *T {
	return p.lastResult
}
