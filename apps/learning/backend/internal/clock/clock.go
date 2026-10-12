package clock

import "time"

// Clock is the time source domain code depends on, so business logic stays
// deterministic in tests instead of calling time.Now directly.
type Clock interface {
	Now() time.Time
}

type realTime struct{}

// New returns a Clock backed by the real wall clock (UTC).
func New() Clock { return realTime{} }

func (realTime) Now() time.Time {
	return time.Now().UTC()
}
