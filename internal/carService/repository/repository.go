package repository

import "context"

type repo struct{}

func New() *repo {
	return &repo{}
}

func (r *repo) GetTestMessage(ctx context.Context) (string, error) {
	return "Hello!", nil
}
