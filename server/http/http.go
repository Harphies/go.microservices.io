package http

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// A Server defines parameters for serve HTTP requests, a wrapper around http.Server
type Server struct {
	logger *zap.Logger
}

func New(logger *zap.Logger, routes http.Handler, port int, wg sync.WaitGroup) error {

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           routes,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	// make a channel to communicate error within processes
	shutDownError := make(chan error)
	go func() {
		// make a channel to receive os signal signals
		quit := make(chan os.Signal, 1) // buffered channel with maximum value of 1

		// listen to SIGINT and SIGTERM signals and put it in quit channel
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// read the signal receive in quit channel into a variable
		receivedSignal := <-quit // this blocks until it receives a SIGINT or SIGTERM signal

		// Log a message when a signal is received and stringify the received signal
		logger.Info("caught signal", zap.String("received signal", receivedSignal.String()))

		// Create a context with 5-second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// once a SIGTERM or SIGINT signal is received, make a call to shut down the signal
		if err := srv.Shutdown(ctx); err != nil {
			shutDownError <- err // put the error in the error channel if error occurs during shutting down the server
		}

		// Wait for all background processes to complete their task
		logger.Info("Waiting for all background processes to complete their task!")

		wg.Wait() // wait for all background goroutines to complete
		shutDownError <- nil
	}()

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// wait to receive the value in the shutdown channel
	err = <-shutDownError
	// check if the shutdown channel returns an error
	if err != nil {
		return err
	}

	// server successfully shut down
	logger.Info("Server successfully shutdown!")

	return nil
}
