package main

import (
	"context"
<<<<<<< HEAD
	httpDelivery "go_project/internal/carService/delivery/http"
	"go_project/internal/carService/repository"
	"go_project/internal/carService/usecase"
	"log"
=======
	"log"
	"log/slog"
>>>>>>> 911d87d (лр 2)
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
<<<<<<< HEAD
)

func main() {
	repo := repository.New()
	uc := usecase.New(repo)
	h := httpDelivery.New(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("/test", h.Test())

	srv := &http.Server{
		Addr:    ":8082",
=======

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"go_project/internal/config"
	"go_project/internal/handler"
	"go_project/internal/repository/postgres"
	"go_project/internal/usecase"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sqlx.Connect("postgres", cfg.DB.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	orderRepo := postgres.NewOrderRepo(db)
	testRepo := postgres.NewTestRepo(db)

	userUC := usecase.NewUserUsecase(userRepo, sessionRepo)
	orderUC := usecase.NewOrderUsecase(orderRepo)
	testUC := usecase.NewTestUsecase(testRepo)

	userH := handler.NewUserHandler(userUC)
	testH := handler.NewTestHandler(testUC)
	_ = orderUC

	mux := http.NewServeMux()
	mux.HandleFunc("/register", userH.Register)
	mux.HandleFunc("/test", testH.Test())
	mux.HandleFunc("/dbtest", testH.DbTest())

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
>>>>>>> 911d87d (лр 2)
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
<<<<<<< HEAD
		log.Println("Server started on :8082")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	<-quit
	log.Println("Shutting down server...")
=======
		slog.Info("server started", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()
	<-quit
	slog.Info("shutting down server")
>>>>>>> 911d87d (лр 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
<<<<<<< HEAD
		log.Fatalf("Shutdown error: %v", err)
	}
	log.Println("Server stopped gracefully")
=======
		slog.Error("shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped gracefully")
>>>>>>> 911d87d (лр 2)
}
