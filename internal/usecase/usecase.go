package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"go_project/internal/domain"
)

type userUsecase struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
}

func NewUserUsecase(ur domain.UserRepository, sr domain.SessionRepository) domain.UserUsecase {
	return &userUsecase{userRepo: ur, sessionRepo: sr}
}

func (u *userUsecase) Register(name, email, password string) (*domain.User, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
	user := &domain.User{
		Login:    name,
		Email:    email,
		Password: hash,
		Phone:    "",
		IsActive: true,
	}
	err := u.userRepo.CreateUser(user)
	return user, err
}

func (u *userUsecase) Login(email, password string) (*domain.Session, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
	user, err := u.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Password != hash {
		return nil, fmt.Errorf("invalid credentials")
	}
	token := fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	session := &domain.Session{
		SessionID: token,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	err = u.sessionRepo.CreateSession(session)
	return session, err
}

type orderUsecase struct {
	orderRepo domain.OrderRepository
}

func NewOrderUsecase(or domain.OrderRepository) domain.OrderUsecase {
	return &orderUsecase{orderRepo: or}
}

func (o *orderUsecase) PlaceOrder(userID string, items []domain.OrderItem) (*domain.Order, error) {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	_ = total
	order := &domain.Order{UserID: userID, Status: "pending"}
	err := o.orderRepo.CreateOrder(order)
	return order, err
}

func (o *orderUsecase) GetOrders(userID string) ([]domain.Order, error) {
	return o.orderRepo.GetOrdersByUserID(userID)
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
