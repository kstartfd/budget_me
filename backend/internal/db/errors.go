package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// Infrastructure error classification for the query layer. Kept next to the
// generated code (like convert.go) so feature packages map database errors the
// same way without importing the pgx driver themselves.

// IsNotFound reports whether err is pgx's no-rows sentinel, i.e. a :one query
// matched nothing. Feature handlers translate it into their own domain
// "not found" error.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
