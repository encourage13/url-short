package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/encourage13/url-short/internal/domain"
)

type InMemoryRepository struct {
	mu   sync.RWMutex
	urls map[string]*domain.URL
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		urls: make(map[string]*domain.URL),
	}
}

func (r *InMemoryRepository) Save(_ context.Context, url *domain.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.urls[url.Code]; exists {
		return domain.ErrCodeExists
	}
	stored := *url
	r.urls[url.Code] = &stored
	return nil
}

func (r *InMemoryRepository) FindByCode(_ context.Context, code string) (*domain.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, exists := r.urls[code]
	if !exists {
		return nil, domain.ErrNotFound
	}
	found := *url
	return &found, nil
}

func (r *InMemoryRepository) IncrementClicks(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	url, exists := r.urls[code]
	if !exists {
		return domain.ErrNotFound
	}

	url.Clicks++
	return nil
}

func (r *InMemoryRepository) List(_ context.Context, limit, offset int) ([]*domain.URL, error) {
	r.mu.RLock()
	urls := make([]*domain.URL, 0, len(r.urls))
	for _, url := range r.urls {
		found := *url
		urls = append(urls, &found)
	}
	r.mu.RUnlock()

	sort.Slice(urls, func(i, j int) bool {
		return urls[i].CreatedAt.After(urls[j].CreatedAt)
	})

	if offset >= len(urls) {
		return []*domain.URL{}, nil
	}
	return urls[offset:min(offset+limit, len(urls))], nil
}

func (r *InMemoryRepository) Ping(context.Context) error {
	return nil
}
