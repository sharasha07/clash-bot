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
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.cfg.Port),
		Handler:      app.routes(),
		ReadTimeout:  app.cfg.Server.ReadTimeout,
		WriteTimeout: app.cfg.Server.WriteTimeout,
		IdleTimeout:  app.cfg.Server.IdleTimeout,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		log.Printf("shutting down server, signal: %s", s.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(ctx)
	}()

	log.Printf("starting server on port: %d", app.cfg.Port)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	log.Println("stopped server")
	return nil
}
