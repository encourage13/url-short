package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type URL struct {
	ID          uuid.UUID
	Code        string
	OriginalURL string
	CreatedAt   time.Time
	Clicks      int64
}

var (
	ErrNotFound   = errors.New("url not found")
	ErrCodeExists = errors.New("code already exists")
	ErrInvalidURL = errors.New("invalid url")
)
