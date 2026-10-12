package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens the pgx connection pool used across the backend.
func NewPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, url)
}
