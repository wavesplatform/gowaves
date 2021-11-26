package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/cmd/netmon/internal"
)

const (
	defaultNetworkTimeout  = 15
	defaultPollingInterval = 60
)

var (
	errInvalidParameters = errors.New("invalid parameters")
)

func main() {
	err := run()
	switch err {
	case context.Canceled:
		os.Exit(130)
	case errInvalidParameters:
		os.Exit(2)
	default:
		os.Exit(1)
	}
}

func run() error {
	var (
		nodesList   string
		bindAddress string
		interval    int
		timeout     int
	)
	flag.StringVar(&nodesList, "nodes", "", "List of Waves Blockchain sample nodes REST APIs")
	flag.StringVar(&bindAddress, "bind", ":8080", "Local network address to bind the HTTP API of the service on. Default value is \":8080\".")
	flag.IntVar(&interval, "interval", defaultPollingInterval, "Polling interval, seconds. Default value is 60")
	flag.IntVar(&timeout, "timeout", defaultNetworkTimeout, "Network timeout, seconds. Default value is 15")
	flag.Parse()

	if len(nodesList) == 0 || len(strings.Fields(nodesList)) < 2 {
		log.Printf("Invalid nodes list '%s'", nodesList)
		return errInvalidParameters
	}
	if interval <= 0 {
		log.Printf("Invalid polling interval '%d'", interval)
		return errInvalidParameters
	}
	if timeout <= 0 {
		log.Printf("Invalid network timout '%d'", timeout)
		return errInvalidParameters
	}

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
	defer done()

	m, err := internal.NewNodeMonitor(nodesList, timeout)
	if err != nil {
		log.Printf("ERROR: Failed to start monitoring: %v", err)
		return err
	}
	m.Start(ctx)

	api, err := internal.NewHealthService(m)
	if err != nil {
		log.Printf("Failed to intialize API: %v", err)
		return err
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Mount("/", routes(api))

	srv := &http.Server{Addr: bindAddress, Handler: r}
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start REST API at '%s': %v", bindAddress, err)
		}
	}()

	<-ctx.Done()
	log.Println("Terminated")
	return nil
}

func routes(api *internal.HealthService) chi.Router {
	r := chi.NewRouter()
	r.Get("/health", api.Health)
	return r
}
