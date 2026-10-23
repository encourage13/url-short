package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/encourage13/url-short/internal/domain"
	"github.com/encourage13/url-short/internal/generator"
	"github.com/encourage13/url-short/internal/repository"
	"github.com/google/uuid"
)

const maxCodeAttempts = 5

type URLService struct {
	repo       repository.URLRepository
	codeLength int
}

func NewURLService(repo repository.URLRepository, codeLength int) *URLService {
	return &URLService{
		repo:       repo,
		codeLength: codeLength,
	}
}

func (s *URLService) CreateShortURL(ctx context.Context, originalURL string) (*domain.URL, error) {
	if err := validateURL(originalURL); err != nil {
		return nil, err
	}

	for attempt := 0; attempt < maxCodeAttempts; attempt++ {
		shortURL := &domain.URL{
			ID:          uuid.New(),
			Code:        generator.GenerateShortCode(s.codeLength),
			OriginalURL: originalURL,
			CreatedAt:   time.Now(),
		}

		err := s.repo.Save(ctx, shortURL)
		if err == nil {
			return shortURL, nil
		}
		if !errors.Is(err, domain.ErrCodeExists) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("could not generate unique code in %d attempts", maxCodeAttempts)
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return domain.ErrInvalidURL
	}
	return nil
}

func (s *URLService) GetOriginalURL(ctx context.Context, code string) (*domain.URL, error) {
	found, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if err := s.repo.IncrementClicks(ctx, code); err != nil {
		return nil, err
	}

	return found, nil
}

func (s *URLService) GetURLInfo(ctx context.Context, code string) (*domain.URL, error) {
	return s.repo.FindByCode(ctx, code)
}

func (s *URLService) ListURLs(ctx context.Context, limit, offset int) ([]*domain.URL, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *URLService) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
