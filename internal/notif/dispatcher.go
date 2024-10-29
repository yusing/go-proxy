package notif

import (
	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	Dispatcher struct {
		task      task.Task
		logCh     chan *LogMessage
		providers F.Set[Provider]
	}
	LogMessage struct {
		Level          zerolog.Level
		Title, Message string
	}
)

var dispatcher *Dispatcher

var ErrUnknownNotifProvider = E.New("unknown notification provider")

const dispatchErr = "notification dispatch error"

func init() {
	dispatcher = newNotifDispatcher()
	go dispatcher.start()
}

func newNotifDispatcher() *Dispatcher {
	return &Dispatcher{
		task:      task.GlobalTask("notif dispatcher"),
		logCh:     make(chan *LogMessage),
		providers: F.NewSet[Provider](),
	}
}

func GetDispatcher() *Dispatcher {
	return dispatcher
}

func RegisterProvider(configSubTask task.Task, cfg ProviderConfig) (Provider, error) {
	name := configSubTask.Name()
	createFunc, ok := Providers[name]
	if !ok {
		return nil, ErrUnknownNotifProvider.
			Subject(name).
			Withf(strutils.DoYouMean(utils.NearestField(name, Providers)))
	}
	if provider, err := createFunc(cfg); err != nil {
		return nil, err
	} else {
		dispatcher.providers.Add(provider)
		configSubTask.OnCancel("remove provider", func() {
			dispatcher.providers.Remove(provider)
		})
		return provider, nil
	}
}

func (disp *Dispatcher) start() {
	defer dispatcher.task.Finish("dispatcher stopped")

	for {
		select {
		case <-disp.task.Context().Done():
			return
		case entry := <-disp.logCh:
			go disp.dispatch(entry)
		}
	}
}

func (disp *Dispatcher) dispatch(msg *LogMessage) {
	task := disp.task.Subtask("dispatch notif")
	defer task.Finish("notif dispatched")

	errs := E.NewBuilder(dispatchErr)
	disp.providers.RangeAllParallel(func(p Provider) {
		if err := p.Send(task.Context(), msg); err != nil {
			errs.Add(E.PrependSubject(p.Name(), err))
		}
	})
	if errs.HasError() {
		E.LogError(errs.About(), errs.Error())
	}
}

// Run implements zerolog.Hook.
// func (disp *Dispatcher) Run(e *zerolog.Event, level zerolog.Level, message string) {
// 	if strings.HasPrefix(message, dispatchErr) { // prevent recursion
// 		return
// 	}
// 	switch level {
// 	case zerolog.WarnLevel, zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
// 		disp.logCh <- &LogMessage{
// 			Level:   level,
// 			Message: message,
// 		}
// 	}
// }

func Notify(title, msg string) {
	dispatcher.logCh <- &LogMessage{
		Level:   zerolog.InfoLevel,
		Title:   title,
		Message: msg,
	}
}
