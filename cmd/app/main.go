package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go_project/internal/config"
	"go_project/internal/handler"
	appmetrics "go_project/internal/metrics"
	"go_project/internal/middleware"
	"go_project/internal/rabbitmq"
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

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	if err := godotenv.Overload(); err != nil {
		log.Warn("no .env file found or error loading; using environment variables", slog.Any("error", err))
	}

	cfg, err := config.New()
	if err != nil {
		log.Error("config error", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := connectPostgres(cfg.DB.DSN(), 30, 500*time.Millisecond)
	if err != nil {
		log.Error("db connect error", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	rabbit, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ, appCtx, log)
	if err != nil {
		log.Error("rabbitmq connect error", slog.Any("error", err))
		os.Exit(1)
	}
	defer rabbit.Close()

	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	orderRepo := postgres.NewOrderRepo(db, rabbit, log)
	testRepo := postgres.NewTestRepo(db)
	orderRepo.StartStatusChangeConsumer(rabbitmq.OrderStatusQueue)

	userUC := usecase.NewUserUsecase(userRepo, sessionRepo)
	orderUC := usecase.NewOrderUsecase(orderRepo, log)
	testUC := usecase.NewTestUsecase(testRepo)

	userH := handler.NewUserHandler(userUC, log)
	orderH := handler.NewOrderHandler(orderUC, log)
	testH := handler.NewTestHandler(testUC, log)
	middlewareManager := middleware.NewMiddlewareManager(log, sessionRepo)
	serverMetrics := appmetrics.NewServerMetrics(prometheus.DefaultRegisterer)

	mux := http.NewServeMux()
	mux.Handle("POST /auth/register", userH.RegisterUser())
	mux.Handle("POST /auth/login", userH.LoginUser())
	mux.Handle("POST /orders/create", middlewareManager.JWTMiddleware(middlewareManager.SessionMiddleware(orderH.AddNewOrder())))
	mux.Handle("GET /orders/list", middlewareManager.JWTMiddleware(middlewareManager.SessionMiddleware(orderH.GetOrdersList())))
	mux.HandleFunc("/register", userH.Register)
	mux.HandleFunc("/test", testH.Test())
	mux.HandleFunc("/dbtest", testH.DbTest())
	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: serverMetrics.Middleware(mux),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server started", slog.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
		os.Exit(1)
	}
	appCancel()
	log.Info("server stopped gracefully")
}
