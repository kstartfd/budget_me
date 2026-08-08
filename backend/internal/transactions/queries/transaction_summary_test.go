package queries

import (
	"testing"
	"time"
)

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

// carryWindowFrom — обрезка окна прошлого периода по точке сброса истории.
// Всё, что до отсечения, считалось по СТАРЫМ границам месяца, поэтому вперёд не
// переносится; граница `to` исключающая (SQL берёт spent_at < to).
func TestCarryWindowFrom(t *testing.T) {
	at := func(d int) time.Time { return time.Date(2026, 8, d, 0, 0, 0, 0, time.UTC) }
	cutoff := at(10)

	tests := []struct {
		name     string
		from, to time.Time
		cutoff   *time.Time
		wantFrom time.Time
		wantOK   bool
	}{
		{"сброса не было — окно как есть", at(1), at(5), nil, at(1), true},
		{"окно целиком до отсечения", at(1), at(5), &cutoff, time.Time{}, false},
		{"окно упирается в отсечение — граница исключающая", at(1), at(10), &cutoff, time.Time{}, false},
		{"окно задевает отсечение — начало подтягивается", at(5), at(15), &cutoff, cutoff, true},
		{"окно целиком после отсечения", at(12), at(20), &cutoff, at(12), true},
		{"окно начинается ровно в отсечение", at(10), at(20), &cutoff, at(10), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotOK := carryWindowFrom(tt.from, tt.to, tt.cutoff)

			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}

			if tt.wantOK && !gotFrom.Equal(tt.wantFrom) {
				t.Errorf("from = %v, want %v", gotFrom, tt.wantFrom)
			}
		})
	}
}
