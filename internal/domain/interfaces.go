package domain

import (
	"context"
	"time"
)

type User struct {
	ID        string    `db:"id"`
	Login     string    `db:"login"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	Phone     string    `db:"phone"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Session struct {
	SessionID string    `db:"session_id"`
	UserID    string    `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}

type Order struct {
	ID          string     `db:"id"`
	UserID      string     `db:"user_id"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
	CompletedAt *time.Time `db:"completed_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

type OrderItem struct {
	Price    float64
	Quantity int
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByID(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
}

type SessionRepository interface {
	CreateSession(session *Session) error
	GetSessionByToken(token string) (*Session, error)
}

type OrderRepository interface {
	CreateOrder(order *Order) error
	GetOrdersByUserID(userID string) ([]Order, error)
}

type TestRepository interface {
	GetTestMessage(ctx context.Context) (string, error)
	SaveMessage(ctx context.Context, message string) error
}

type UserUsecase interface {
	Register(name, email, password string) (*User, error)
	Login(email, password string) (*Session, error)
}

type OrderUsecase interface {
	PlaceOrder(userID string, items []OrderItem) (*Order, error)
	GetOrders(userID string) ([]Order, error)
}

type TestUsecase interface {
	GetTestMessage(ctx context.Context) (string, error)
	SaveMessage(ctx context.Context, message string) error
}
