// Filename: cmd/api/server.go

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {

	//create our http server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// shutdown function should return its error to this channel
	shutdownError := make(chan error)

	// create a background goroutine
	go func() {
		// create a quit/exit channel which carries os.Signal values
		quit := make(chan os.Signal, 1)
		// listen for SIGINT and SIGTERM signals
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Block until a signal is received
		s := <-quit

		// log message
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})
		// create a context with a 20 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		//call shutdown function
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		// Log message about the goroutines
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})
		app.wg.Wait()
		shutdownError <- nil
	}()
	// start our server
	app.logger.PrintInfo("Starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})
	// check if shutdown process has been triggered
	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// block for notification from shutdown function
	err = <-shutdownError
	if err != nil {
		return err
	}

	// graceful shutdown was successful
	// start our server
	app.logger.PrintInfo("gracefully shutdown server", map[string]string{
		"addr": srv.Addr,
	})
	return nil

}
