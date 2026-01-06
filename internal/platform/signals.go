//go:build !windows
// +build !windows

package platform

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/pidfile"
)

// ExitFunc is a function type for exiting the program. It can be replaced with a mock function in tests.
type ExitFunc func(code int)

// SignalHandler encapsulates signal handling dependencies to make the code more testable.
type SignalHandler struct {
	exitFunc ExitFunc
}

// NewSignalHandler creates a new SignalHandler instance.
// If exitFunc is nil, it uses the default os.Exit.
func NewSignalHandler(exitFunc ExitFunc) *SignalHandler {
	if exitFunc == nil {
		exitFunc = os.Exit
	}
	return &SignalHandler{
		exitFunc: exitFunc,
	}
}

// SetupSignals sets up the signal handler and returns the created signal channel.
// If the provided signals is nil, a new channel is created.
func SetupSignals(signals chan os.Signal, reloadFn func(), pidFile *pidfile.PIDFile) chan os.Signal {
	return SetupSignalsWithHandler(signals, reloadFn, pidFile, nil)
}

// SetupSignalsWithHandler sets up the signal handler with support for custom ExitFunc for testing.
func SetupSignalsWithHandler(signals chan os.Signal, reloadFn func(), pidFile *pidfile.PIDFile, exitFunc ExitFunc) chan os.Signal {
	logger.Infof("setting up os signal watcher")

	if signals == nil {
		signals = make(chan os.Signal, 1)
	}
	signal.Notify(signals, syscall.SIGUSR1)
	signal.Notify(signals, syscall.SIGHUP)
	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, os.Interrupt)

	handler := NewSignalHandler(exitFunc)
	go handler.watchForSignals(signals, reloadFn, pidFile)

	return signals
}

// watchForSignals listens for signals and handles them.
func (h *SignalHandler) watchForSignals(signals chan os.Signal, reloadFn func(), pidFile *pidfile.PIDFile) {
	logger.Info("os signal watcher ready")

	for {
		sig := <-signals
		switch sig {
		case syscall.SIGUSR1:
			logger.Info("caught USR1 signal")
			reloadFn()

		case syscall.SIGHUP:
			logger.Info("caught HUP signal")
			reloadFn()

		case os.Interrupt, syscall.SIGTERM:
			logger.Infof("caught %s signal; exiting", sig)
			if pidFile != nil {
				err := pidFile.Remove()
				if err != nil {
					logger.Error(fmt.Sprintf("%v", err))
				}
			}
			h.exitFunc(0)

		default:
			logger.Warnf("caught unhandled signal %+v", sig)
		}
	}
}
