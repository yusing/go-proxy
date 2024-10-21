package notif

import (
	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/task"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	Dispatcher struct {
		task      task.Task
		logCh     chan *logrus.Entry
		providers F.Set[Provider]
	}
)

var dispatcher *Dispatcher

func init() {
	dispatcher = newNotifDispatcher()
	go dispatcher.start()
}

func newNotifDispatcher() *Dispatcher {
	return &Dispatcher{
		task:      task.GlobalTask("notif dispatcher"),
		logCh:     make(chan *logrus.Entry),
		providers: F.NewSet[Provider](),
	}
}

func GetDispatcher() *Dispatcher {
	return dispatcher
}

func RegisterProvider(configSubTask task.Task, cfg ProviderConfig) (Provider, E.Error) {
	name := configSubTask.Name()
	createFunc, ok := Providers[name]
	if !ok {
		return nil, E.NotExist("provider", name)
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
	defer close(dispatcher.logCh)

	for {
		select {
		case <-disp.task.Context().Done():
			return
		case entry := <-disp.logCh:
			go disp.dispatch(entry)
		}
	}
}

func (disp *Dispatcher) dispatch(entry *logrus.Entry) {
	task := disp.task.Subtask("dispatch notif")
	defer task.Finish("notifs dispatched")

	errs := E.NewBuilder("errors sending notif")
	disp.providers.RangeAllParallel(func(p Provider) {
		if err := p.Send(task.Context(), entry); err != nil {
			errs.Addf("%s: %s", p.Name(), err)
		}
	})
	if err := errs.Build(); err != nil {
		logrus.Error("notif dispatcher failure: ", err)
	}
}

// Levels implements logrus.Hook.
func (disp *Dispatcher) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// Fire implements logrus.Hook.
func (disp *Dispatcher) Fire(entry *logrus.Entry) error {
	if disp.providers.Size() == 0 {
		return nil
	}
	disp.logCh <- entry
	return nil
}
