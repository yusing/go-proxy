package provider

import (
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher"
)

type EventHandler struct {
	provider *Provider

	added   []string
	removed []string
	paused  []string
	updated []string
	errs    E.Builder
}

func (provider *Provider) newEventHandler() *EventHandler {
	return &EventHandler{
		provider: provider,
		errs:     E.NewBuilder("event errors"),
	}
}

func (handler *EventHandler) Handle(parent task.Task, events []watcher.Event) {
	oldRoutes := handler.provider.routes
	newRoutes, err := handler.provider.LoadRoutesImpl()
	if err != nil {
		handler.errs.Add(err.Subject("load routes"))
		if newRoutes.Size() == 0 {
			return
		}
	}

	if common.IsDebug {
		eventsLog := E.NewBuilder("events")
		for _, event := range events {
			eventsLog.Addf("event %s, actor: name=%s, id=%s", event.Action, event.ActorName, event.ActorID)
		}
		handler.provider.l.Debug(eventsLog.String())
		oldRoutesLog := E.NewBuilder("old routes")
		oldRoutes.RangeAll(func(k string, r *route.Route) {
			oldRoutesLog.Addf(k)
		})
		handler.provider.l.Debug(oldRoutesLog.String())
		newRoutesLog := E.NewBuilder("new routes")
		newRoutes.RangeAll(func(k string, r *route.Route) {
			newRoutesLog.Addf(k)
		})
		handler.provider.l.Debug(newRoutesLog.String())
	}

	oldRoutes.RangeAll(func(k string, oldr *route.Route) {
		newr, ok := newRoutes.Load(k)
		if !ok {
			handler.Remove(oldr)
		} else if handler.matchAny(events, newr) {
			handler.Update(parent, oldr, newr)
		} else if entry.ShouldNotServe(newr) {
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
	case ProviderTypeDocker:
		return route.Entry.Container.ContainerID == event.ActorID ||
			route.Entry.Container.ContainerName == event.ActorName
	case ProviderTypeFile:
		return true
	}
	// should never happen
	return false
}

func (handler *EventHandler) Add(parent task.Task, route *route.Route) {
	err := handler.provider.startRoute(parent, route)
	if err != nil {
		handler.errs.Add(E.FailWith("add "+route.Entry.Alias, err))
	} else {
		handler.added = append(handler.added, route.Entry.Alias)
	}
}

func (handler *EventHandler) Remove(route *route.Route) {
	route.Finish("route removed")
	handler.provider.routes.Delete(route.Entry.Alias)
	handler.removed = append(handler.removed, route.Entry.Alias)
}

func (handler *EventHandler) Update(parent task.Task, oldRoute *route.Route, newRoute *route.Route) {
	oldRoute.Finish("route update")
	err := handler.provider.startRoute(parent, newRoute)
	if err != nil {
		handler.errs.Add(E.FailWith("update "+newRoute.Entry.Alias, err))
	} else {
		handler.updated = append(handler.updated, newRoute.Entry.Alias)
	}
}

func (handler *EventHandler) Log() {
	results := E.NewBuilder("event occured")
	for _, alias := range handler.added {
		results.Addf("added %s", alias)
	}
	for _, alias := range handler.removed {
		results.Addf("removed %s", alias)
	}
	for _, alias := range handler.updated {
		results.Addf("updated %s", alias)
	}
	results.Add(handler.errs.Build())
	if result := results.Build(); result != nil {
		handler.provider.l.Info(result)
	}
}
