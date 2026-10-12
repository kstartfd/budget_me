package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Conversion helpers between Go pointers and the pgtype values sqlc uses for
// nullable columns. Kept next to the generated code so every feature package
// maps rows the same way (sqlc only rewrites its own files, so this survives
// regeneration).

// Text wraps an optional string as a nullable pg text.
func Text(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *s, Valid: true}
}

// TextPtr unwraps a nullable pg text to an optional string.
func TextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}

	return &t.String
}

// Int8 wraps an optional int64 as a nullable pg int8.
func Int8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: *v, Valid: true}
}

// Int8Ptr unwraps a nullable pg int8 to an optional int64.
func Int8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}

	return &v.Int64
}

// Time wraps a time as a non-null pg timestamptz.
func Time(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// TimePtr unwraps a nullable pg timestamptz to an optional time, normalised to
// UTC (the zone the whole backend speaks).
func TimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}

	utc := t.Time.UTC()

	return &utc
}
