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

// ToPgOpt wraps an OPTIONAL uuid: uuid.Nil becomes SQL NULL, anything else a
// present value. Mirror of FromPg, which already maps null → uuid.Nil — use it
// for nullable columns / sqlc.narg params.
func ToPgOpt(u uuid.UUID) pgtype.UUID {
	if u == uuid.Nil {
		return pgtype.UUID{}
	}

	return ToPg(u)
}

// FromPg unwraps a pgtype UUID, returning uuid.Nil when it is null.
func FromPg(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}

	return uuid.UUID(p.Bytes)
}
