package events

import (
	"time"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/task"
)

type (
	EventQueue struct {
		task          task.Task
		queue         []Event
		ticker        *time.Ticker
		flushInterval time.Duration
		onFlush       OnFlushFunc
		onError       OnErrorFunc
	}
	OnFlushFunc = func(flushTask task.Task, events []Event)
	OnErrorFunc = func(err E.Error)
)

const eventQueueCapacity = 10

// NewEventQueue returns a new EventQueue with the given
// queueTask, flushInterval, onFlush and onError.
//
// The returned EventQueue will start a goroutine to flush events in the queue
// when the flushInterval is reached.
//
// The onFlush function is called when the flushInterval is reached and the queue is not empty,
//
// The onError function is called when an error received from the errCh,
// or panic occurs in the onFlush function. Panic will cause a E.ErrPanicRecv error.
//
// flushTask.Finish must be called after the flush is done,
// but the onFlush function can return earlier (e.g. run in another goroutine).
//
// If task is canceled before the flushInterval is reached, the events in queue will be discarded.
func NewEventQueue(parent task.Task, flushInterval time.Duration, onFlush OnFlushFunc, onError OnErrorFunc) *EventQueue {
	return &EventQueue{
		task:          parent.Subtask("event queue"),
		queue:         make([]Event, 0, eventQueueCapacity),
		ticker:        time.NewTicker(flushInterval),
		flushInterval: flushInterval,
		onFlush:       onFlush,
		onError:       onError,
	}
}

func (e *EventQueue) Start(eventCh <-chan Event, errCh <-chan E.Error) {
	go func() {
		defer e.ticker.Stop()
		for {
			select {
			case <-e.task.Context().Done():
				return
			case <-e.ticker.C:
				if len(e.queue) > 0 {
					flushTask := e.task.Subtask("flush events")
					queue := e.queue
					e.queue = make([]Event, 0, eventQueueCapacity)
					if !common.IsDebug {
						go func() {
							defer func() {
								if err := recover(); err != nil {
									e.onError(E.Errorf("recovered panic in onFlush: %v", err).Subject(e.task.Parent().String()))
								}
							}()
							e.onFlush(flushTask, queue)
						}()
					} else {
						go e.onFlush(flushTask, queue)
					}
					flushTask.Wait()
				}
				e.ticker.Reset(e.flushInterval)
			case event, ok := <-eventCh:
				e.queue = append(e.queue, event)
				if !ok {
					return
				}
			case err := <-errCh:
				if err != nil {
					e.onError(err)
				}
			}
		}
	}()
}

// Wait waits for all events to be flushed and the task to finish.
//
// It is safe to call this method multiple times.
func (e *EventQueue) Wait() {
	e.task.Wait()
}
