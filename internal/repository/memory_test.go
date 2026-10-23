package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/encourage13/url-short/internal/domain"
)

func TestInMemoryRepository_List(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	base := time.Now()
	for i := 0; i < 5; i++ {
		repo.Save(ctx, &domain.URL{Code: fmt.Sprint(i), CreatedAt: base.Add(time.Duration(i) * time.Minute)})
	}

	tests := []struct {
		limit, offset int
		want          []string
	}{
		{2, 0, []string{"4", "3"}},
		{2, 2, []string{"2", "1"}},
		{2, 4, []string{"0"}},
		{2, 10, []string{}},
	}
	for _, tt := range tests {
		urls, err := repo.List(ctx, tt.limit, tt.offset)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]string, len(urls))
		for i, u := range urls {
			got[i] = u.Code
		}
		if fmt.Sprint(got) != fmt.Sprint(tt.want) {
			t.Errorf("List(%d, %d) = %v, want %v", tt.limit, tt.offset, got, tt.want)
		}
	}
}
