package db

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// dateLayout is the wire format for a plain calendar date everywhere in this
// API ("2026-09-12", no time, no zone) — the client's local day, same
// convention as settings' `today`.
const dateLayout = "2006-01-02"

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

// Int2 wraps an optional int (e.g. a 1..10 rating) as a nullable pg int2. The
// caller validates the range — this is a pure width conversion.
func Int2(v *int) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}

	return pgtype.Int2{Int16: int16(*v), Valid: true}
}

// Int2Ptr unwraps a nullable pg int2 to an optional int.
func Int2Ptr(v pgtype.Int2) *int {
	if !v.Valid {
		return nil
	}

	n := int(v.Int16)

	return &n
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

// ParseDate parses a wire date ("2026-09-12") into a non-null pg date. Callers
// at an HTTP edge should shape-check with a regexp first (matching the
// project's convention for other wire strings) — this still returns an error
// so a calendar-invalid date like "2026-02-30" is caught too.
func ParseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("invalid date %q: %w", s, err)
	}

	return pgtype.Date{Time: t, Valid: true}, nil
}

// ParseDateOpt is ParseDate for an OPTIONAL date: nil stays SQL NULL.
func ParseDateOpt(s *string) (pgtype.Date, error) {
	if s == nil {
		return pgtype.Date{}, nil
	}

	return ParseDate(*s)
}

// DateStr formats a pg date back to the wire format, "" when null.
func DateStr(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}

	return d.Time.Format(dateLayout)
}

// DateStrPtr is DateStr for an optional column — nil when null.
func DateStrPtr(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}

	s := d.Time.Format(dateLayout)

	return &s
}

// ParseDirection maps a wire string ("income"/"expense") to a TxDirection. The
// bool is false for any other value, so HTTP handlers can reject an unknown
// direction at the edge. Kept next to the generated code so every feature maps
// the enum the same way.
func ParseDirection(s string) (TxDirection, bool) {
	switch TxDirection(s) {
	case TxDirectionExpense, TxDirectionIncome:
		return TxDirection(s), true
	default:
		return "", false
	}
}

// ParseVerdict maps a wire string to an ItemVerdict. Mirrors ParseDirection.
func ParseVerdict(s string) (ItemVerdict, bool) {
	switch ItemVerdict(s) {
	case ItemVerdictWorks, ItemVerdictMeh, ItemVerdictNo, ItemVerdictUnknown:
		return ItemVerdict(s), true
	default:
		return "", false
	}
}
