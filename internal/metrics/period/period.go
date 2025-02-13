package period

import (
	"sync"
	"time"
)

type Period[T any] struct {
	FiveMinutes    *Entries[T]
	FifteenMinutes *Entries[T]
	OneHour        *Entries[T]
	OneDay         *Entries[T]
	OneMonth       *Entries[T]
	mu             sync.RWMutex
}

type Filter string

const (
	PeriodFiveMinutes    Filter = "5m"
	PeriodFifteenMinutes Filter = "15m"
	PeriodOneHour        Filter = "1h"
	PeriodOneDay         Filter = "1d"
	PeriodOneMonth       Filter = "1mo"
)

func NewPeriod[T any]() *Period[T] {
	return &Period[T]{
		FiveMinutes:    newEntries[T](5 * time.Minute),
		FifteenMinutes: newEntries[T](15 * time.Minute),
		OneHour:        newEntries[T](1 * time.Hour),
		OneDay:         newEntries[T](24 * time.Hour),
		OneMonth:       newEntries[T](30 * 24 * time.Hour),
	}
}

func (p *Period[T]) Add(info *T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	p.FifteenMinutes.Add(now, info)
	p.OneHour.Add(now, info)
	p.OneDay.Add(now, info)
	p.OneMonth.Add(now, info)
}

func (p *Period[T]) Get(filter Filter) []*T {
	p.mu.RLock()
	defer p.mu.RUnlock()
	switch filter {
	case PeriodFiveMinutes:
		return p.FiveMinutes.Get()
	case PeriodFifteenMinutes:
		return p.FifteenMinutes.Get()
	case PeriodOneHour:
		return p.OneHour.Get()
	case PeriodOneDay:
		return p.OneDay.Get()
	case PeriodOneMonth:
		return p.OneMonth.Get()
	default:
		panic("invalid period filter")
	}
}

func (filter Filter) IsValid() bool {
	switch filter {
	case PeriodFiveMinutes, PeriodFifteenMinutes, PeriodOneHour, PeriodOneDay, PeriodOneMonth:
		return true
	}
	return false
}
