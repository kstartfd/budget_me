package queries

import "testing"

// deltaPct — единственная арифметика, которую сводка добавляет к сырым суммам:
// процент изменения расхода категории против прошлого периода. Ноль-база обязана
// давать nil («нет бейджа»), а не делить на ноль; знак и округление — по месту.
func TestDeltaPct(t *testing.T) {
	tests := []struct {
		name string
		cur  int64
		prev int64
		want *int32
	}{
		{"spent half again as much", 15000, 10000, ptr(50)},
		{"spent a third less", 6667, 10000, ptr(-33)}, // -33.33 → -33
		{"unchanged", 10000, 10000, ptr(0)},
		{"rounds half away from zero", 12550, 10000, ptr(26)}, // 25.5 → 26
		{"dropped to nothing", 0, 10000, ptr(-100)},
		{"no base — prev is zero", 5000, 0, nil},
		{"no base — prev is negative", 5000, -1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deltaPct(tt.cur, tt.prev)

			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("deltaPct(%d, %d) = %d, want nil", tt.cur, tt.prev, *got)
			case tt.want != nil && got == nil:
				t.Fatalf("deltaPct(%d, %d) = nil, want %d", tt.cur, tt.prev, *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Errorf("deltaPct(%d, %d) = %d, want %d", tt.cur, tt.prev, *got, *tt.want)
			}
		})
	}
}

func ptr(v int32) *int32 { return &v }

// carriedInMinor — арифметика переноса неизрасходованного прихода: max(приход −
// расход, 0) за прошлый период. Перерасход прошлого периода обязан клэмпиться в
// 0, а не переноситься долгом.
func TestCarriedInMinor(t *testing.T) {
	tests := []struct {
		name   string
		income int64
		spent  int64
		want   int64
	}{
		{"leftover carries in full", 15000, 10000, 5000},
		{"fully spent — nothing to carry", 10000, 10000, 0},
		{"overspent — clamps to zero, not a debt", 8000, 10000, 0},
		{"no income at all", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := carriedInMinor(tt.income, tt.spent); got != tt.want {
				t.Errorf("carriedInMinor(%d, %d) = %d, want %d", tt.income, tt.spent, got, tt.want)
			}
		})
	}
}
