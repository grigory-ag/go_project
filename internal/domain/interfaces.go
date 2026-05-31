package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string    `json:"id,omitempty" db:"id"`
	Login     string    `json:"login" db:"login"`
	Password  string    `json:"-" db:"password"`
	Email     string    `json:"email,omitempty" db:"email"`
	Phone     string    `json:"phone,omitempty" db:"phone"`
	IsActive  bool      `json:"is_active,omitempty" db:"is_active"`
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

type Session struct {
	SessionID string    `json:"sessionID" db:"session_id"`
	UserID    string    `json:"userID" db:"user_id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`
}

type Order struct {
	ID          string     `db:"id"`
	UserID      string     `db:"user_id"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
	CompletedAt *time.Time `db:"completed_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

type NewOrderData struct {
	UserID string `db:"user_id" json:"-" validate:"required"`
	Amount int    `db:"amount" json:"amount" validate:"required,min=1"`
}

type OrderInfo struct {
	ID        string    `db:"id" json:"orderID"`
	Status    string    `db:"status" json:"status"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByID(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByLogin(ctx context.Context, login string) (*User, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByToken(token string) (*Session, error)
	GetSessionByUserID(ctx context.Context, userID string) (*Session, error)
	UpdateSessionExpiry(ctx context.Context, sessionID string) error
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID string) (uuid.UUID, error)
	GetOrdersForUser(ctx context.Context, userID string, isActive bool) ([]*OrderInfo, error)
	PublishNewOrder(orderID string) error
}

type TestRepository interface {
	GetTestMessage(ctx context.Context) (string, error)
	SaveMessage(ctx context.Context, message string) error
}

type UserUsecase interface {
	RegisterUser(ctx context.Context, login, password string) (*User, error)
	GetUserByLogin(ctx context.Context, login string) (*User, error)
	CreateSession(ctx context.Context, sessionID, userID string) error
}

type OrderUsecase interface {
	CreateOrder(ctx context.Context, userID string) (uuid.UUID, error)
	GetOrders(ctx context.Context, userID string, isActive bool) ([]*OrderInfo, error)
}

type TestUsecase interface {
	GetTestMessage(ctx context.Context) (string, error)
	SaveMessage(ctx context.Context, message string) error
}
