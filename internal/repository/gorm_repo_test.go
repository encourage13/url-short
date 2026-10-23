package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/encourage13/url-short/db/migrations"
	"github.com/encourage13/url-short/internal/domain"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestGormRepo(t *testing.T) *GormRepository {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN не задан")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })

	migrator, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrator.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	return NewGormRepository(db)
}

func TestGormRepository(t *testing.T) {
	repo := newTestGormRepo(t)
	ctx := context.Background()

	code := uuid.NewString()[:10]
	url := &domain.URL{ID: uuid.New(), Code: code, OriginalURL: "https://example.com", CreatedAt: time.Now()}
	t.Cleanup(func() { repo.db.Where("code = ?", code).Delete(&urlModel{}) })

	if err := repo.Save(ctx, url); err != nil {
		t.Fatal(err)
	}

	dup := &domain.URL{ID: uuid.New(), Code: code, OriginalURL: "https://other.example", CreatedAt: time.Now()}
	if err := repo.Save(ctx, dup); !errors.Is(err, domain.ErrCodeExists) {
		t.Errorf("Save duplicate: got %v, want ErrCodeExists", err)
	}

	if err := repo.IncrementClicks(ctx, code); err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindByCode(ctx, code)
	if err != nil {
		t.Fatal(err)
	}
	if found.OriginalURL != url.OriginalURL || found.Clicks != 1 {
		t.Errorf("FindByCode = %+v", found)
	}

	if _, err := repo.FindByCode(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("FindByCode missing: got %v, want ErrNotFound", err)
	}
	if err := repo.IncrementClicks(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("IncrementClicks missing: got %v, want ErrNotFound", err)
	}
	if err := repo.Ping(ctx); err != nil {
		t.Errorf("Ping: %v", err)
	}
}
