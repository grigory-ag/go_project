package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go_project/internal/config"
	"go_project/internal/rabbitmq"
	workerrepository "go_project/internal/worker_service/repository"
	workerservice "go_project/internal/worker_service/service"

	"github.com/joho/godotenv"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	if err := godotenv.Overload(); err != nil {
		log.Warn("no .env file found or error loading; using environment variables", slog.Any("error", err))
	}

	cfg, err := config.LoadWorkersConfig()
	if err != nil {
		log.Error("failed to load worker config", slog.Any("error", err))
		return
	}

	termCtx, termCancel := context.WithCancel(context.Background())
	defer termCancel()
	go waitSigterm(termCancel, log)

	rabbitCtx, rabbitCancel := context.WithCancel(context.Background())
	defer rabbitCancel()

	rabbit, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ, rabbitCtx, log)
	if err != nil {
		log.Error("failed to create RabbitMQ connection", slog.Any("error", err))
		return
	}
	defer rabbit.Close()

	orderCh := make(chan string, 100)
	serviceRepo := workerrepository.NewServiceRepo(rabbit, orderCh, termCtx, log)
	serviceRepo.StartNewOrdersConsumer(rabbitmq.NewOrdersQueue)

	workerPool := workerservice.NewWorkerPool(cfg.Workers, 100, serviceRepo, log)
	workerPool.Run(termCtx, orderCh)

	rabbitCancel()
	log.Info("worker service terminated")
}

func waitSigterm(terminate context.CancelFunc, log *slog.Logger) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	caughtSignal := <-sigCh
	log.Warn("worker service starts termination", slog.String("signal", caughtSignal.String()))

	signal.Stop(sigCh)
	close(sigCh)
	terminate()
}
