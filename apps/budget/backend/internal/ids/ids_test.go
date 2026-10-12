package ids

import (
	"testing"

	"github.com/google/uuid"
)

// ToPgOpt is what nullable-uuid params (sqlc.narg) ride on: uuid.Nil MUST become
// SQL NULL, not an all-zeros value that would then be matched as a real id.
func TestToPgOpt(t *testing.T) {
	if got := ToPgOpt(uuid.Nil); got.Valid {
		t.Errorf("ToPgOpt(uuid.Nil).Valid = true, want NULL")
	}

	u := uuid.New()

	got := ToPgOpt(u)
	if !got.Valid || uuid.UUID(got.Bytes) != u {
		t.Errorf("ToPgOpt(%v) = %+v, want present %v", u, got, u)
	}
}
