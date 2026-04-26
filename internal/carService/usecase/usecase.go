package usecase

import (
	"context"
	carService "go_project/internal/carService"
)

type useCase struct {
	repo carService.Repository
}

func New(repo carService.Repository) *useCase {
	return &useCase{repo: repo}
}

func (uc *useCase) GetTestMessage(ctx context.Context) (string, error) {
	return uc.repo.GetTestMessage(ctx)
}
