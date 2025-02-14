package period

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type (
	PollFunc[T any]                  func(ctx context.Context, lastResult *T) (*T, error)
	AggregateFunc[T, AggregateT any] func(entries []*T, query url.Values) (total int, result AggregateT)
	FilterFunc[T any]                func(entries []*T, keyword string) (filtered []*T)
	Poller[T, AggregateT any]        struct {
		name         string
		poll         PollFunc[T]
		aggregator   AggregateFunc[T, AggregateT]
		resultFilter FilterFunc[T]
		period       *Period[T]
		lastResult   *T
		errs         []pollErr
	}
	pollErr struct {
		err   error
		count int
	}
)

const (
	pollInterval       = 1 * time.Second
	gatherErrsInterval = 30 * time.Second
)

func NewPoller[T any](
	name string,
	poll PollFunc[T],
) *Poller[T, T] {
	return &Poller[T, T]{
		name:   name,
		poll:   poll,
		period: NewPeriod[T](),
	}
}

func NewPollerWithAggregator[T, AggregateT any](
	name string,
	poll PollFunc[T],
	aggregator AggregateFunc[T, AggregateT],
) *Poller[T, AggregateT] {
	return &Poller[T, AggregateT]{
		name:       name,
		poll:       poll,
		aggregator: aggregator,
		period:     NewPeriod[T](),
	}
}

func (p *Poller[T, AggregateT]) WithResultFilter(filter FilterFunc[T]) *Poller[T, AggregateT] {
	p.resultFilter = filter
	return p
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
	errs := gperr.NewBuilder(fmt.Sprintf("poller %s has encountered %d errors in the last %s:", p.name, len(p.errs), gatherErrsInterval))
	for _, e := range p.errs {
		errs.Addf("%w: %d times", e.err, e.count)
	}
	return errs.String(), true
}

func (p *Poller[T, AggregateT]) clearErrs() {
	p.errs = p.errs[:0]
}

func (p *Poller[T, AggregateT]) pollWithTimeout(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, pollInterval)
	defer cancel()
	data, err := p.poll(ctx, p.lastResult)
	if err != nil {
		p.appendErr(err)
		return
	}
	p.period.Add(data)
	p.lastResult = data
}

func (p *Poller[T, AggregateT]) Start() {
	go func() {
		ctx := task.RootContext()
		ticker := time.NewTicker(pollInterval)
		gatherErrsTicker := time.NewTicker(gatherErrsInterval)
		defer ticker.Stop()
		defer gatherErrsTicker.Stop()

		logging.Debug().Msgf("Starting poller %s with interval %s", p.name, pollInterval)

		p.pollWithTimeout(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.pollWithTimeout(ctx)
			case <-gatherErrsTicker.C:
				errs, ok := p.gatherErrs()
				if ok {
					logging.Error().Msg(errs)
				}
				p.clearErrs()
			}
		}
	}()
}

func (p *Poller[T, AggregateT]) Get(filter Filter) ([]*T, bool) {
	return p.period.Get(filter)
}

func (p *Poller[T, AggregateT]) GetLastResult() *T {
	return p.lastResult
}
