package notif

import (
	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/config/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
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
		Level  zerolog.Level
		Title  string
		Extras map[string]any
		Color  Color
	}
)

var dispatcher *Dispatcher

var (
	ErrMissingNotifProvider     = E.New("missing notification provider")
	ErrInvalidNotifProviderType = E.New("invalid notification provider type")
	ErrUnknownNotifProvider     = E.New("unknown notification provider")
)

const dispatchErr = "notification dispatch error"

func StartNotifDispatcher(parent task.Task) *Dispatcher {
	dispatcher = &Dispatcher{
		task:      parent.Subtask("notification dispatcher"),
		logCh:     make(chan *LogMessage),
		providers: F.NewSet[Provider](),
	}
	go dispatcher.start()
	return dispatcher
}

func Notify(msg *LogMessage) {
	if dispatcher == nil {
		return
	}
	dispatcher.logCh <- msg
}

func (disp *Dispatcher) RegisterProvider(cfg types.NotificationConfig) (Provider, E.Error) {
	providerName, ok := cfg["provider"]
	if !ok {
		return nil, ErrMissingNotifProvider
	}
	switch providerName := providerName.(type) {
	case string:
		delete(cfg, "provider")
		createFunc, ok := Providers[providerName]
		if !ok {
			return nil, ErrUnknownNotifProvider.
				Subject(providerName).
				Withf(strutils.DoYouMean(utils.NearestField(providerName, Providers)))
		}

		provider, err := createFunc(cfg)
		if err == nil {
			disp.providers.Add(provider)
		}
		return provider, err
	default:
		return nil, ErrInvalidNotifProviderType.Subjectf("%T", providerName)
	}
}

func (disp *Dispatcher) start() {
	defer func() {
		disp.providers.Clear()
		close(disp.logCh)
		disp.task.Finish("dispatcher stopped")
	}()

	for {
		select {
		case <-disp.task.Context().Done():
			return
		case msg := <-disp.logCh:
			go disp.dispatch(msg)
		}
	}
}

func (disp *Dispatcher) dispatch(msg *LogMessage) {
	task := disp.task.Subtask("dispatch notif")
	defer task.Finish("notif dispatched")

	errs := E.NewBuilder(dispatchErr)
	disp.providers.RangeAllParallel(func(p Provider) {
		if err := notifyProvider(task.Context(), p, msg); err != nil {
			errs.Add(E.PrependSubject(p.Name(), err))
		}
	})
	if errs.HasError() {
		E.LogError(errs.About(), errs.Error())
	} else {
		logging.Debug().Str("title", msg.Title).Msgf("dispatched notif")
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
