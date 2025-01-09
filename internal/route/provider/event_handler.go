package provider

import (
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/route/entry"
	"github.com/yusing/go-proxy/internal/route/provider/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher"
)

type EventHandler struct {
	provider *Provider

	errs    *E.Builder
	added   *E.Builder
	removed *E.Builder
	updated *E.Builder
}

func (p *Provider) newEventHandler() *EventHandler {
	return &EventHandler{
		provider: p,
		errs:     E.NewBuilder("event errors"),
		added:    E.NewBuilder("added"),
		removed:  E.NewBuilder("removed"),
		updated:  E.NewBuilder("updated"),
	}
}

func (handler *EventHandler) Handle(parent task.Parent, events []watcher.Event) {
	oldRoutes := handler.provider.routes
	newRoutes, err := handler.provider.loadRoutesImpl()
	if err != nil {
		handler.errs.Add(err)
		if newRoutes.Size() == 0 {
			return
		}
	}

	if common.IsDebug {
		eventsLog := E.NewBuilder("events")
		for _, event := range events {
			eventsLog.Addf("event %s, actor: name=%s, id=%s", event.Action, event.ActorName, event.ActorID)
		}
		E.LogDebug(eventsLog.About(), eventsLog.Error(), handler.provider.Logger())

		oldRoutesLog := E.NewBuilder("old routes")
		oldRoutes.RangeAllParallel(func(k string, r *route.Route) {
			oldRoutesLog.Adds(k)
		})
		E.LogDebug(oldRoutesLog.About(), oldRoutesLog.Error(), handler.provider.Logger())

		newRoutesLog := E.NewBuilder("new routes")
		newRoutes.RangeAllParallel(func(k string, r *route.Route) {
			newRoutesLog.Adds(k)
		})
		E.LogDebug(newRoutesLog.About(), newRoutesLog.Error(), handler.provider.Logger())
	}

	oldRoutes.RangeAll(func(k string, oldr *route.Route) {
		newr, ok := newRoutes.Load(k)
		switch {
		case !ok:
			handler.Remove(oldr)
		case handler.matchAny(events, newr):
			handler.Update(parent, oldr, newr)
		case entry.ShouldNotServe(newr):
			handler.Remove(oldr)
		}
	})
	newRoutes.RangeAll(func(k string, newr *route.Route) {
		if !(oldRoutes.Has(k) || entry.ShouldNotServe(newr)) {
			handler.Add(parent, newr)
		}
	})
}

func (handler *EventHandler) matchAny(events []watcher.Event, route *route.Route) bool {
	for _, event := range events {
		if handler.match(event, route) {
			return true
		}
	}
	return false
}

func (handler *EventHandler) match(event watcher.Event, route *route.Route) bool {
	switch handler.provider.GetType() {
	case types.ProviderTypeDocker:
		return route.Entry.Container.ContainerID == event.ActorID ||
			route.Entry.Container.ContainerName == event.ActorName
	case types.ProviderTypeFile:
		return true
	}
	// should never happen
	return false
}

func (handler *EventHandler) Add(parent task.Parent, route *route.Route) {
	err := handler.provider.startRoute(parent, route)
	if err != nil {
		handler.errs.Add(err.Subject("add"))
	} else {
		handler.added.Adds(route.Entry.Alias)
	}
}

func (handler *EventHandler) Remove(route *route.Route) {
	route.Finish("route removed")
	handler.provider.routes.Delete(route.Entry.Alias)
	handler.removed.Adds(route.Entry.Alias)
}

func (handler *EventHandler) Update(parent task.Parent, oldRoute *route.Route, newRoute *route.Route) {
	oldRoute.Finish("route update")
	err := handler.provider.startRoute(parent, newRoute)
	if err != nil {
		handler.errs.Add(err.Subject("update"))
	} else {
		handler.updated.Adds(newRoute.Entry.Alias)
	}
}

func (handler *EventHandler) Log() {
	results := E.NewBuilder("event occurred")
	results.AddFrom(handler.added, false)
	results.AddFrom(handler.removed, false)
	results.AddFrom(handler.updated, false)
	results.AddFrom(handler.errs, false)
	if result := results.String(); result != "" {
		handler.provider.Logger().Info().Msg(result)
	}
}
