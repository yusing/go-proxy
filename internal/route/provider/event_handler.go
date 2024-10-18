package provider

import (
	E "github.com/yusing/go-proxy/internal/error"
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
		return
	}

	oldRoutes.RangeAll(func(k string, v *route.Route) {
		if !newRoutes.Has(k) {
			handler.Remove(v)
		}
	})
	newRoutes.RangeAll(func(k string, newr *route.Route) {
		if oldRoutes.Has(k) {
			for _, ev := range events {
				if handler.match(ev, newr) {
					old, ok := oldRoutes.Load(k)
					if !ok { // should not happen
						panic("race condition")
					}
					handler.Update(parent, old, newr)
					return
				}
			}
		} else {
			handler.Add(parent, newr)
		}
	})
}

func (handler *EventHandler) match(event watcher.Event, route *route.Route) bool {
	switch handler.provider.t {
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
		handler.errs.Add(err)
	} else {
		handler.added = append(handler.added, route.Entry.Alias)
	}
}

func (handler *EventHandler) Remove(route *route.Route) {
	route.Finish("route removal")
	handler.removed = append(handler.removed, route.Entry.Alias)
}

func (handler *EventHandler) Update(parent task.Task, oldRoute *route.Route, newRoute *route.Route) {
	oldRoute.Finish("route update")
	err := handler.provider.startRoute(parent, newRoute)
	if err != nil {
		handler.errs.Add(err)
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
