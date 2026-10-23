package repository

import (
	"context"

	"github.com/encourage13/url-short/internal/domain"
)

type URLRepository interface {
	Save(ctx context.Context, url *domain.URL) error

	FindByCode(ctx context.Context, code string) (*domain.URL, error)

	IncrementClicks(ctx context.Context, code string) error

	List(ctx context.Context, limit, offset int) ([]*domain.URL, error)

	Ping(ctx context.Context) error
}
