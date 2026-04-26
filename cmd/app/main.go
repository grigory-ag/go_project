package main

import (
	"context"
	httpDelivery "go_project/internal/carService/delivery/http"
	"go_project/internal/carService/repository"
	"go_project/internal/carService/usecase"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	repo := repository.New()
	uc := usecase.New(repo)
	h := httpDelivery.New(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("/test", h.Test())

	srv := &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server started on :8082")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	log.Println("Server stopped gracefully")
}
