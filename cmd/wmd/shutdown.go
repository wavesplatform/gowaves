package main

import (
	"go.uber.org/zap"
	"os"
	"os/signal"
)

var (
	interruptSignals       = []os.Signal{os.Interrupt}
	shutdownRequestChannel = make(chan struct{})
)

func interruptListener(log *zap.SugaredLogger) <-chan struct{} {
	r := make(chan struct{})

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, interruptSignals...)
		select {
		case sig := <-signals:
			log.Infof("Caught signal '%s', shutting down...", sig)
		case <-shutdownRequestChannel:
			log.Info("Shutdown requested, shutting down...")
		}
		close(r)
		for {
			select {
			case sig := <-signals:
				log.Infof("Caught signal '%s' again, already shutting down", sig)
			case <-shutdownRequestChannel:
				log.Info("Repetitive shutdown request, already shutting down")
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
