package transactions

import "testing"

// Старый клиент не шлёт counted — трата должна участвовать в расчёте, как и вела
// себя всегда; только явный false выводит её из расчёта.
func TestRequestCountedDefaultsToTrue(t *testing.T) {
	yes, no := true, false

	cases := []struct {
		name string
		in   *bool
		want bool
	}{
		{"missing", nil, true},
		{"true", &yes, true},
		{"false", &no, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (transactionRequest{Counted: c.in}).counted(); got != c.want {
				t.Errorf("counted() = %v, want %v", got, c.want)
			}
		})
	}
}
