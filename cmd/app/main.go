package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	db, err := sqlx.Connect("postgres", cfg.DB.DSN())
	if err != nil {
		slog.Error("db connect error", "err", err)
		os.Exit(1)
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
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server started", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped gracefully")
}
