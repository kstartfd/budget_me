package placement_test

import (
	"testing"

	"learning_app/backend/internal/learningpath/placement"
)

func TestLadderEdges(t *testing.T) {
	if next, ok := placement.Next(placement.A1); !ok || next != placement.A2 {
		t.Fatalf("above A1: %q %t", next, ok)
	}

	if _, ok := placement.Next(placement.C2); ok {
		t.Fatal("the ladder must end at C2")
	}

	if below := placement.Below(placement.A1); below != placement.A1 {
		t.Fatalf("below A1 must clamp at A1, got %q", below)
	}

	if below := placement.Below(placement.C2); below != placement.C1 {
		t.Fatalf("below C2: %q", below)
	}
}

func TestGradeForgivesCaseAndEdges(t *testing.T) {
	q := placement.Question{Answer: "had I closed"}

	if !placement.Grade(q, "  Had  I closed ") {
		t.Fatal("case and spacing must not fail a right answer")
	}

	if placement.Grade(q, "I closed") {
		t.Fatal("a wrong option graded right")
	}
}
