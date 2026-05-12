package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go_project/internal/domain"

	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(u *domain.User) error {
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

type sessionRepo struct {
	db *sqlx.DB
}

func NewSessionRepo(db *sqlx.DB) domain.SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) CreateSession(s *domain.Session) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (session_id, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.SessionID, s.UserID, s.ExpiresAt,
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

type orderRepo struct {
	db *sqlx.DB
}

func NewOrderRepo(db *sqlx.DB) domain.OrderRepository {
	return &orderRepo{db: db}
}

func (r *orderRepo) CreateOrder(o *domain.Order) error {
	return r.db.QueryRowx(
		`INSERT INTO orders (user_id, status) VALUES ($1, $2)
		 RETURNING id, created_at, completed_at, updated_at`,
		o.UserID, o.Status,
	).StructScan(o)
}

func (r *orderRepo) GetOrdersByUserID(userID string) ([]domain.Order, error) {
	var orders []domain.Order
	if err := r.db.Select(&orders, `SELECT * FROM orders WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}
	return orders, nil
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
