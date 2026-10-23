package repository

import (
	"context"
	"errors"
	"time"

	"github.com/encourage13/url-short/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type urlModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        string
	OriginalURL string
	CreatedAt   time.Time
	Clicks      int64
}

func (urlModel) TableName() string { return "urls" }

func toModel(u *domain.URL) *urlModel {
	return &urlModel{ID: u.ID, Code: u.Code, OriginalURL: u.OriginalURL, CreatedAt: u.CreatedAt, Clicks: u.Clicks}
}

func (m *urlModel) toDomain() *domain.URL {
	return &domain.URL{ID: m.ID, Code: m.Code, OriginalURL: m.OriginalURL, CreatedAt: m.CreatedAt, Clicks: m.Clicks}
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Save(ctx context.Context, url *domain.URL) error {
	err := r.db.WithContext(ctx).Create(toModel(url)).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrCodeExists
	}
	return err
}

func (r *GormRepository) FindByCode(ctx context.Context, code string) (*domain.URL, error) {
	var m urlModel
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return m.toDomain(), nil
}

func (r *GormRepository) IncrementClicks(ctx context.Context, code string) error {
	result := r.db.WithContext(ctx).Model(&urlModel{}).Where("code = ?", code).Update("clicks", gorm.Expr("clicks + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) List(ctx context.Context, limit, offset int) ([]*domain.URL, error) {
	var models []urlModel
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}

	urls := make([]*domain.URL, len(models))
	for i := range models {
		urls[i] = models[i].toDomain()
	}
	return urls, nil
}

func (r *GormRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
