package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"go_project/internal/domain"

	"github.com/google/uuid"
)

type userUsecase struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
}

func NewUserUsecase(ur domain.UserRepository, sr domain.SessionRepository) domain.UserUsecase {
	return &userUsecase{userRepo: ur, sessionRepo: sr}
}

func (u *userUsecase) RegisterUser(ctx context.Context, login, password string) (*domain.User, error) {
	existing, err := u.userRepo.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("user with login %s already exists", login)
	}

	user := &domain.User{
		Login:    login,
		Password: password,
		Phone:    "",
		IsActive: true,
	}

	if err := u.userRepo.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userUsecase) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	return u.userRepo.GetUserByLogin(ctx, login)
}

func (u *userUsecase) CreateSession(ctx context.Context, sessionID, userID string) error {
	session := &domain.Session{
		SessionID: sessionID,
		UserID:    userID,
	}
	return u.sessionRepo.CreateSession(ctx, session)
}

type orderUsecase struct {
	orderRepo domain.OrderRepository
	log       *slog.Logger
}

func NewOrderUsecase(or domain.OrderRepository) domain.OrderUsecase {
	return &orderUsecase{orderRepo: or, log: slog.Default()}
}

func (o *orderUsecase) CreateOrder(ctx context.Context, userID string) (uuid.UUID, error) {
	orderID, err := o.orderRepo.CreateOrder(ctx, userID)
	if err != nil {
		o.log.Error("Failed to create order", slog.Any("error", err))
		return uuid.UUID{}, err
	}

	if err := o.orderRepo.PublishNewOrder(orderID.String()); err != nil {
		o.log.Error("Failed to publish new order", slog.Any("error", err))
		return uuid.UUID{}, err
	}

	return orderID, nil
}

func (o *orderUsecase) GetOrders(ctx context.Context, userID string, isActive bool) ([]*domain.OrderInfo, error) {
	orders, err := o.orderRepo.GetOrdersForUser(ctx, userID, isActive)
	if err != nil {
		o.log.Error("Failed to get orders for user", slog.Any("userID", userID), slog.Any("error", err))
		return nil, err
	}

	return orders, nil
}

type testUsecase struct {
	repo domain.TestRepository
}

func NewTestUsecase(repo domain.TestRepository) domain.TestUsecase {
	return &testUsecase{repo: repo}
}

func (t *testUsecase) GetTestMessage(ctx context.Context) (string, error) {
	return t.repo.GetTestMessage(ctx)
}

func (t *testUsecase) SaveMessage(ctx context.Context, message string) error {
	return t.repo.SaveMessage(ctx, message)
}
