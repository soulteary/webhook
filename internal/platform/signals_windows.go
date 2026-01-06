//go:build windows
// +build windows

package platform

import (
	"os"
	"os/signal"

	"github.com/soulteary/webhook/internal/logger"
	"github.com/soulteary/webhook/internal/pidfile"
)

// SetupSignals sets up the signal handler and returns the created signal channel.
// On Windows, this handles Ctrl+C (SIGINT) and SIGTERM.
func SetupSignals(signals chan os.Signal, reloadFn func(), pidFile *pidfile.PIDFile) chan os.Signal {
	return SetupSignalsWithShutdown(signals, reloadFn, nil, pidFile, nil)
}

// SetupSignalsWithShutdown sets up the signal handler with support for shutdown callback.
func SetupSignalsWithShutdown(signals chan os.Signal, reloadFn func(), shutdownFn func(), pidFile *pidfile.PIDFile, exitFunc ExitFunc) chan os.Signal {
	return SetupSignalsWithHandlerAndShutdown(signals, reloadFn, shutdownFn, pidFile, exitFunc)
}

// SetupSignalsWithHandler sets up the signal handler with support for custom ExitFunc for testing.
func SetupSignalsWithHandler(signals chan os.Signal, reloadFn func(), pidFile *pidfile.PIDFile, exitFunc ExitFunc) chan os.Signal {
	return SetupSignalsWithHandlerAndShutdown(signals, reloadFn, nil, pidFile, exitFunc)
}

// SetupSignalsWithHandlerAndShutdown sets up the signal handler with support for shutdown callback and custom ExitFunc.
func SetupSignalsWithHandlerAndShutdown(signals chan os.Signal, reloadFn func(), shutdownFn func(), pidFile *pidfile.PIDFile, exitFunc ExitFunc) chan os.Signal {
	logger.Infof("setting up os signal watcher")

	if signals == nil {
		signals = make(chan os.Signal, 1)
	}
	signal.Notify(signals, os.Interrupt)

	handler := NewSignalHandler(exitFunc)
	go handler.watchForSignals(signals, reloadFn, shutdownFn, pidFile)

	return signals
}
