package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"go_project/internal/config"
	"go_project/internal/handler"
	appjwt "go_project/internal/jwt"
	"go_project/internal/repository/postgres"
	"go_project/internal/usecase"
)

func connectPostgres(dsn string, attempts int, delay time.Duration) (*sqlx.DB, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		db, err := sqlx.Connect("postgres", dsn)
		if err == nil {
			return db, nil
		}
		lastErr = err
		if i < attempts {
			slog.Warn("database not ready, retrying", "attempt", i, "max", attempts, "err", err)
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
		claims, err := appjwt.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

func main() {
	if err := godotenv.Overload(); err != nil {
		log.Fatal(err)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.New()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	db, err := connectPostgres(cfg.DB.DSN(), 30, 500*time.Millisecond)
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
	orderH := handler.NewOrderHandler(orderUC)
	testH := handler.NewTestHandler(testUC)

	mux := http.NewServeMux()
	mux.Handle("POST /auth/register", userH.RegisterUser())
	mux.Handle("POST /auth/login", userH.LoginUser())
	mux.Handle("POST /orders/create", withAuth(orderH.AddNewOrder()))
	mux.Handle("GET /orders/list", withAuth(orderH.GetOrdersList()))
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
