package main

import (
	"log/slog"
	"os"
	"os/signal"
)

var (
	interruptSignals       = []os.Signal{os.Interrupt}
	shutdownRequestChannel = make(chan struct{})
)

func interruptListener() <-chan struct{} {
	r := make(chan struct{})

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, interruptSignals...)
		select {
		case sig := <-signals:
			slog.Info("Caught signal, shutting down...", "signal", sig)
		case <-shutdownRequestChannel:
			slog.Info("Shutdown requested, shutting down...")
		}
		close(r)
		for {
			select {
			case sig := <-signals:
				slog.Info("Caught signal again, already shutting down", "signal", sig)
			case <-shutdownRequestChannel:
				slog.Info("Repetitive shutdown request, already shutting down")
			}
		}
	}()
	return r
}

func interruptRequested(interrupted <-chan struct{}) bool {
	select {
	case <-interrupted:
		return true
	default:
	}
	return false
}
