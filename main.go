package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/tappsi/airbrake-webhook/webhook"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// main obtains the configuration, starts the connection pool for the messaging queue,
// registers a cleanup handler, starts the web server, registers a handler for the endpoint
// and starts listening.
func main() {

	cfg := webhook.LoadConfiguration("./config/")
	queue := webhook.NewMessagingQueue(cfg.QueueURI, cfg.ExchangeName, cfg.PoolConfig)
	hook := webhook.NewWebHook(queue)

	router := newRouter(cfg.EndpointName, http.HandlerFunc(hook.Process))
	go cleanup(queue)

	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.WebServerPort), router)
	webhook.FailOnError(err, "Error listening on web server")

}

// newRouter creates a new http router for handling requests, receives as parameters
// the endpoint name and the http handler.
func newRouter(endpoint string, handler http.Handler) *mux.Router {

	router := mux.NewRouter()

	router.StrictSlash(true).
		Methods("POST").
		Path("/" + endpoint).
		Name("AirbrakeWebhook").
		Handler(handler)

	return router

}

// cleanup creates a channel for receiving system signals, when an interrupt is
// received it stops the connection pool to the queue and stops the web server.
// It receives as parameter a MessagingQueue.
func cleanup(queue webhook.MessagingQueue) {

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGTSTP)
	<-sigChan

	fmt.Println("\nReceived an interrupt, stopping services...\n")
	queue.Close()

	runtime.GC()
	os.Exit(0)

}
