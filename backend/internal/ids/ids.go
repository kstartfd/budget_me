// Package ids mints application identifiers and converts them to and from the
// pgtype values sqlc uses at the database boundary.
package ids

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// New returns a fresh time-ordered UUIDv7.
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// ToPg wraps a uuid.UUID as the pgtype value sqlc expects.
func ToPg(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// FromPg unwraps a pgtype UUID, returning uuid.Nil when it is null.
func FromPg(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}

	return uuid.UUID(p.Bytes)
}
