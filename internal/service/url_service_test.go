package service

import (
	"context"
	"errors"
	"testing"

	"github.com/encourage13/url-short/internal/domain"
	"github.com/encourage13/url-short/internal/repository"
)

func TestCreateShortURL_InvalidURL(t *testing.T) {
	svc := NewURLService(repository.NewInMemoryRepository(), 6)

	for _, raw := range []string{"hello", "javascript:alert(1)", "ftp://example.com", "http://", "/relative"} {
		if _, err := svc.CreateShortURL(context.Background(), raw); !errors.Is(err, domain.ErrInvalidURL) {
			t.Errorf("CreateShortURL(%q): got %v, want ErrInvalidURL", raw, err)
		}
	}
}

func TestCreateShortURL_DoesNotOverwriteOnCollision(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryRepository()
	svc := NewURLService(repo, 1)

	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, c := range alphabet {
		if err := repo.Save(ctx, &domain.URL{Code: string(c), OriginalURL: "https://old.example"}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := svc.CreateShortURL(ctx, "https://new.example"); err == nil {
		t.Fatal("expected error when all codes are taken")
	}

	urls, _ := repo.List(ctx, 100, 0)
	for _, u := range urls {
		if u.OriginalURL != "https://old.example" {
			t.Fatalf("code %q was overwritten with %q", u.Code, u.OriginalURL)
		}
	}
}
