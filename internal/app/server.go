package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (a *App) Run(addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           a.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("shutdown signal received")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("Server is listening on http://localhost%s", addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}

	if a.cleanup != nil {
		a.cleanup()
	}

	if a.usersGRPCConn != nil {
		_ = a.usersGRPCConn.Close()
	}

	if a.DB != nil {
		return a.DB.Close()
	}

	return nil
}
