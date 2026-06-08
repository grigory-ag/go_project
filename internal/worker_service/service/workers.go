package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go_project/internal/worker_service"
)

type WorkerPool struct {
	repo         worker_service.Repository
	sem          chan struct{}
	orderQueue   chan string
	queueWg      sync.WaitGroup
	maxWorkers   int
	maxQueueLen  int
	processed    atomic.Int64
	enqueued     atomic.Int64
	rabbitMutex  *sync.Mutex
	shutdownOnce sync.Once
	log          *slog.Logger
}

func NewWorkerPool(maxWorkers, maxQueueLen int, repository worker_service.Repository, log *slog.Logger) *WorkerPool {
	wp := &WorkerPool{
		repo:        repository,
		sem:         make(chan struct{}, maxWorkers),
		orderQueue:  make(chan string, maxQueueLen),
		maxWorkers:  maxWorkers,
		maxQueueLen: maxQueueLen,
		rabbitMutex: &sync.Mutex{},
		log:         log,
	}

	for i := 0; i < maxWorkers; i++ {
		wp.queueWg.Add(1)
		go wp.worker(i)
	}

	return wp
}

func (wp *WorkerPool) worker(id int) {
	defer wp.queueWg.Done()

	for orderID := range wp.orderQueue {
		wp.sem <- struct{}{}

		sleepTime := time.Duration(rand.Intn(4)+1) * time.Second
		wp.log.Info(
			"worker started processing order",
			slog.Int("worker_id", id),
			slog.String("order_id", orderID),
			slog.Duration("processing_time", sleepTime),
			slog.Int("queue_len", len(wp.orderQueue)),
		)

		ctx, cancel := context.WithTimeout(context.Background(), sleepTime)
		<-ctx.Done()

		orderStatus := "CANCELED"
		if ctx.Err() == context.DeadlineExceeded {
			orderStatus = "COMPLETED"
			wp.processed.Add(1)
		}
		cancel()

		wp.rabbitMutex.Lock()
		err := wp.repo.PublishOrderStatus(orderID, orderStatus)
		wp.rabbitMutex.Unlock()
		if err != nil {
			wp.log.Error(
				"failed to publish order status",
				slog.Int("worker_id", id),
				slog.String("order_id", orderID),
				slog.String("status", orderStatus),
				slog.Any("error", err),
			)
		} else {
			wp.log.Info(
				"worker completed order",
				slog.Int("worker_id", id),
				slog.String("order_id", orderID),
				slog.String("status", orderStatus),
			)
		}

		<-wp.sem
	}
}

func (wp *WorkerPool) Run(ctx context.Context, queue chan string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case orderID, ok := <-queue:
			if !ok {
				wp.log.Info("external order queue closed")
				wp.Shutdown()
				return
			}
			if err := wp.ProcessOrder(ctx, orderID); err != nil {
				wp.log.Error(
					"failed to enqueue order",
					slog.String("order_id", orderID),
					slog.Any("error", err),
				)
			}
		case <-ticker.C:
			maxWorkers, currentWorkers, queueLen, processed, enqueued := wp.Stats()
			wp.log.Info(
				"worker pool stats",
				slog.Int64("max_workers", maxWorkers),
				slog.Int64("current_workers", currentWorkers),
				slog.Int64("queue_len", queueLen),
				slog.Int64("processed", processed),
				slog.Int64("enqueued", enqueued),
			)
		case <-ctx.Done():
			wp.log.Info("worker pool context canceled")
			wp.Shutdown()
			return
		}
	}
}

func (wp *WorkerPool) ProcessOrder(ctx context.Context, orderID string) error {
	select {
	case wp.orderQueue <- orderID:
		wp.enqueued.Add(1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("worker queue is full, maximum length is %d", wp.maxQueueLen)
	}
}

func (wp *WorkerPool) Stats() (maxWorkers, currentWorkers, queueLen, processed, enqueued int64) {
	return int64(wp.maxWorkers),
		int64(len(wp.sem)),
		int64(len(wp.orderQueue)),
		wp.processed.Load(),
		wp.enqueued.Load()
}

func (wp *WorkerPool) Shutdown() {
	wp.shutdownOnce.Do(func() {
		close(wp.orderQueue)
		wp.queueWg.Wait()
		wp.log.Info("worker pool stopped gracefully")
	})
}
