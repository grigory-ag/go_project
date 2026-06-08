package repository

import (
	"context"
	"encoding/json"
	"log/slog"

	"go_project/internal/rabbitmq"
	"go_project/internal/worker_service"
	"go_project/internal/worker_service/models"
)

type serviceRepo struct {
	rabbit  *rabbitmq.RabbitMQ
	orderCh chan string
	ctx     context.Context
	log     *slog.Logger
}

func NewServiceRepo(rabbit *rabbitmq.RabbitMQ, orderCh chan string, ctx context.Context, log *slog.Logger) worker_service.Repository {
	return &serviceRepo{
		rabbit:  rabbit,
		orderCh: orderCh,
		ctx:     ctx,
		log:     log,
	}
}

func (r *serviceRepo) PublishOrderStatus(orderID, newStatus string) error {
	body, err := json.Marshal(models.NewStatus{
		OrderID:   orderID,
		NewStatus: newStatus,
	})
	if err != nil {
		return err
	}

	if err := r.rabbit.Publish(rabbitmq.OrderStatusRoutingKey, body); err != nil {
		r.log.Error(
			"failed to publish order status",
			slog.String("order_id", orderID),
			slog.String("status", newStatus),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

func (r *serviceRepo) StartNewOrdersConsumer(queueName string) {
	msgs, err := r.rabbit.Channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		r.log.Error("failed to start new orders consumer", slog.Any("error", err))
		return
	}

	go func() {
		r.log.Info("new orders consumer started", slog.String("queue", queueName))
		for {
			select {
			case <-r.ctx.Done():
				r.log.Info("new orders consumer stopped")
				return
			case delivery, ok := <-msgs:
				if !ok {
					r.log.Error("new orders RabbitMQ channel closed")
					return
				}

				var orderID string
				if err := json.Unmarshal(delivery.Body, &orderID); err != nil {
					r.log.Error("failed to parse new order message", slog.Any("error", err))
					_ = delivery.Nack(false, true)
					continue
				}

				select {
				case r.orderCh <- orderID:
					if err := delivery.Ack(false); err != nil {
						r.log.Error("failed to acknowledge new order message", slog.Any("error", err))
					}
				case <-r.ctx.Done():
					_ = delivery.Nack(false, true)
					return
				}
			}
		}
	}()
}
