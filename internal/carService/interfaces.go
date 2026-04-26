package carService

import (
	"context"
	"net/http"
)

type Handler interface {
	Test() http.HandlerFunc
}

type UseCase interface {
	GetTestMessage(ctx context.Context) (string, error)
}

type Repository interface {
	GetTestMessage(ctx context.Context) (string, error)
}
