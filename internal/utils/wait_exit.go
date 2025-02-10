package utils

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

func WaitExit(shutdownTimeout int) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	// wait for signal
	<-sig

	// gracefully shutdown
	logging.Info().Msg("shutting down")
	_ = task.GracefulShutdown(time.Second * time.Duration(shutdownTimeout))
}
