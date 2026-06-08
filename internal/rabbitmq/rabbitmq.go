package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"go_project/internal/config"

	"github.com/cenkalti/backoff/v4"
	"github.com/streadway/amqp"
)

const (
	MainExchange          = "main_exchange"
	NewOrdersQueue        = "new-orders"
	OrderStatusQueue      = "order-status-change"
	NewOrdersRoutingKey   = "new-orders"
	OrderStatusRoutingKey = "order-status-change"
)

type publisherCfg struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
	QueueName  string
}

type RabbitMQ struct {
	conn          *amqp.Connection
	Channel       *amqp.Channel
	Ctx           context.Context
	PubConf       publisherCfg
	backoffPolicy backoff.BackOff
	publishMu     sync.Mutex
	closeOnce     sync.Once
	log           *slog.Logger
}

func NewRabbitMQ(cfg *config.RabbitMQConfig, ctx context.Context, log *slog.Logger) (*RabbitMQ, error) {
	dsn := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		url.PathEscape(cfg.Vhost),
	)

	policy := backoff.NewExponentialBackOff()
	policy.MaxElapsedTime = 30 * time.Second

	var conn *amqp.Connection
	err := backoff.Retry(func() error {
		select {
		case <-ctx.Done():
			return backoff.Permanent(ctx.Err())
		default:
		}

		var dialErr error
		conn, dialErr = amqp.Dial(dsn)
		if dialErr != nil {
			log.Warn("failed to connect to RabbitMQ, retrying", slog.Any("error", dialErr))
		}
		return dialErr
	}, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	rabbit := &RabbitMQ{
		conn:          conn,
		Channel:       ch,
		Ctx:           ctx,
		backoffPolicy: policy,
		PubConf: publisherCfg{
			Name:      MainExchange,
			Kind:      "direct",
			Durable:   true,
			QueueName: NewOrdersQueue,
		},
		log: log,
	}

	if err := rabbit.declareTopology(); err != nil {
		rabbit.Close()
		return nil, err
	}

	go func() {
		<-ctx.Done()
		rabbit.Close()
	}()

	return rabbit, nil
}

func (r *RabbitMQ) declareTopology() error {
	if err := r.Channel.ExchangeDeclare(
		MainExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	for _, binding := range []struct {
		queue      string
		routingKey string
	}{
		{queue: NewOrdersQueue, routingKey: NewOrdersRoutingKey},
		{queue: OrderStatusQueue, routingKey: OrderStatusRoutingKey},
	} {
		queue, err := r.Channel.QueueDeclare(binding.queue, true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", binding.queue, err)
		}
		if err := r.Channel.QueueBind(queue.Name, binding.routingKey, MainExchange, false, nil); err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", binding.queue, err)
		}
	}

	return nil
}

func (r *RabbitMQ) Publish(routingKey string, body []byte) error {
	r.publishMu.Lock()
	defer r.publishMu.Unlock()

	return r.Channel.Publish(
		MainExchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (r *RabbitMQ) Close() {
	r.closeOnce.Do(func() {
		if r.Channel != nil {
			if err := r.Channel.Close(); err != nil && err != amqp.ErrClosed {
				r.log.Error("failed to close RabbitMQ channel", slog.Any("error", err))
			}
		}
		if r.conn != nil {
			if err := r.conn.Close(); err != nil && err != amqp.ErrClosed {
				r.log.Error("failed to close RabbitMQ connection", slog.Any("error", err))
			}
		}
	})
}
