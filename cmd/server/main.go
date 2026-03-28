package main

import (
	"context"
	"errors"
	"homeward/internal/control"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := ":8080"
	server := control.NewServer(addr)

	// Listen for OS shutdown signals in a separate goroutine.
	// This lets main() block on Start() below without missing signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("forced shutdown: %v", err)
		}
	}()

	log.Printf("control plane listening on %s", addr)
	if err := server.Start(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}

	log.Println("server stopped")
}
