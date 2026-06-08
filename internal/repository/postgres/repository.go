package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"go_project/internal/domain"
	"go_project/internal/rabbitmq"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const (
	createOrderQuery = `INSERT INTO orders (user_id, status) VALUES ($1, 'UNDEFINED') RETURNING id`
	getOrdersQuery   = `SELECT id, status, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC`
	getActiveOrdersQuery = `SELECT id, status, updated_at
		FROM orders
		WHERE user_id = $1 AND status in ('UNDEFINED', 'PACKING', 'ARRIVING')
		ORDER BY created_at DESC`
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(u *domain.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = string(hashedPassword)

	return r.db.QueryRowx(
		`INSERT INTO users (login, password, email, phone, is_active)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		u.Login, u.Password, u.Email, u.Phone, u.IsActive,
	).StructScan(u)
}

func (r *userRepo) GetUserByID(id string) (*domain.User, error) {
	var u domain.User
	if err := r.db.Get(&u, `SELECT * FROM users WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetUserByEmail(email string) (*domain.User, error) {
	var u domain.User
	if err := r.db.Get(&u, `SELECT * FROM users WHERE email = $1`, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	var u domain.User
	if err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE login = $1`, login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

type sessionRepo struct {
	db *sqlx.DB
}

func NewSessionRepo(db *sqlx.DB) domain.SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) CreateSession(ctx context.Context, s *domain.Session) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO sessions (session_id, user_id, created_at, expires_at)
		 VALUES ($1, $2, NOW(), NOW() + INTERVAL '24 hours')`,
		s.SessionID, s.UserID,
	)
	return err
}

func (r *sessionRepo) GetSessionByToken(token string) (*domain.Session, error) {
	var s domain.Session
	if err := r.db.Get(&s, `SELECT * FROM sessions WHERE session_id = $1`, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) GetSessionByUserID(ctx context.Context, userID string) (*domain.Session, error) {
	var s domain.Session
	if err := r.db.GetContext(ctx, &s, `SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) UpdateSessionExpiry(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET expires_at = NOW() + INTERVAL '24 hours' WHERE session_id = $1`, sessionID)
	return err
}

type orderRepo struct {
	db     *sqlx.DB
	rabbit *rabbitmq.RabbitMQ
	log    *slog.Logger
}

func NewOrderRepo(db *sqlx.DB, rabbit *rabbitmq.RabbitMQ, log *slog.Logger) domain.OrderRepository {
	return &orderRepo{db: db, rabbit: rabbit, log: log}
}

func (r *orderRepo) CreateOrder(ctx context.Context, userID string) (uuid.UUID, error) {
	var newOrderID uuid.UUID

	err := r.db.QueryRowContext(ctx, createOrderQuery, userID).Scan(&newOrderID)
	if err != nil {
		return uuid.UUID{}, err
	}

	return newOrderID, nil
}

func (r *orderRepo) GetOrdersForUser(ctx context.Context, userID string, isActive bool) ([]*domain.OrderInfo, error) {
	query := getOrdersQuery
	if isActive {
		query = getActiveOrdersQuery
	}

	rows, err := r.db.QueryxContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.OrderInfo
	for rows.Next() {
		var order domain.OrderInfo
		if err := rows.StructScan(&order); err != nil {
			return nil, fmt.Errorf("failed to scan orders for client rows: %w", err)
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate orders rows: %w", err)
	}

	return orders, nil
}

func (r *orderRepo) PublishNewOrder(orderID string) error {
	body, err := json.Marshal(orderID)
	if err != nil {
		return fmt.Errorf("failed to marshal new order: %w", err)
	}

	if err := r.rabbit.Publish(rabbitmq.NewOrdersRoutingKey, body); err != nil {
		r.log.Error("failed to publish new order", slog.String("order_id", orderID), slog.Any("error", err))
		return err
	}

	r.log.Info("new order published", slog.String("order_id", orderID))
	return nil
}

func (r *orderRepo) StartStatusChangeConsumer(queueName string) {
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
		r.log.Error("failed to start order status consumer", slog.Any("error", err))
		return
	}

	go func() {
		r.log.Info("order status consumer started", slog.String("queue", queueName))
		for {
			select {
			case <-r.rabbit.Ctx.Done():
				r.log.Info("order status consumer stopped")
				return
			case delivery, ok := <-msgs:
				if !ok {
					r.log.Error("order status RabbitMQ channel closed")
					return
				}

				var statusData domain.ChangeOrderStatusData
				if err := json.Unmarshal(delivery.Body, &statusData); err != nil {
					r.log.Error("failed to parse order status message", slog.Any("error", err))
					_ = delivery.Nack(false, true)
					continue
				}

				if err := r.ChangeOrderStatus(r.rabbit.Ctx, &statusData); err != nil {
					r.log.Error(
						"failed to update order status",
						slog.String("order_id", statusData.OrderID),
						slog.String("status", statusData.NewStatus),
						slog.Any("error", err),
					)
					_ = delivery.Nack(false, true)
					continue
				}

				if err := delivery.Ack(false); err != nil {
					r.log.Error("failed to acknowledge order status message", slog.Any("error", err))
					continue
				}

				r.log.Info(
					"order status updated",
					slog.String("order_id", statusData.OrderID),
					slog.String("status", statusData.NewStatus),
				)
			}
		}
	}()
}

func (r *orderRepo) ChangeOrderStatus(ctx context.Context, statusData *domain.ChangeOrderStatusData) error {
	switch statusData.NewStatus {
	case "UNDEFINED", "PACKING", "ARRIVING", "COMPLETED", "CANCELED":
	default:
		return fmt.Errorf("unsupported order status %q", statusData.NewStatus)
	}

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE orders
		 SET status = CAST($2 AS VARCHAR(20)),
		     updated_at = NOW(),
		     completed_at = CASE
		         WHEN CAST($2 AS VARCHAR(20)) = 'COMPLETED' THEN NOW()
		         ELSE completed_at
		     END
		 WHERE id = $1`,
		statusData.OrderID,
		statusData.NewStatus,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("order %s not found", statusData.OrderID)
	}

	return nil
}

type testRepo struct {
	db *sqlx.DB
}

func NewTestRepo(db *sqlx.DB) domain.TestRepository {
	return &testRepo{db: db}
}

func (r *testRepo) GetTestMessage(ctx context.Context) (string, error) {
	var msg string
	err := r.db.GetContext(ctx, &msg, `SELECT message FROM dbtest_messages ORDER BY created_at DESC LIMIT 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "Hello!", nil
		}
		return "", err
	}
	return msg, nil
}

func (r *testRepo) SaveMessage(ctx context.Context, message string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO dbtest_messages (message) VALUES ($1)`, message)
	return err
}
