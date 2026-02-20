package engine

import (
	"testing"

	"dndsl/internal/parser"
)

func TestRollBasic(t *testing.T) {
	expr := &parser.DiceExpr{
		Raw: "3d6",
	}

	res, err := Roll(expr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res.RawRolls) != 3 {
		t.Fatalf("expected 3 raw rolls, got %d", len(res.RawRolls))
	}

	for _, v := range res.RawRolls {
		if v < 1 || v > 6 {
			t.Errorf("roll out of bounds for d6: %d", v)
		}
	}
}

func TestRollAdvantage(t *testing.T) {
	expr := &parser.DiceExpr{
		Raw: "1d20a",
	}

	res, err := Roll(expr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res.RawRolls) != 2 {
		t.Fatalf("advantage should roll 2 dice, got %d", len(res.RawRolls))
	}

	if len(res.Kept) != 1 {
		t.Fatalf("advantage should keep 1 die, got %d", len(res.Kept))
	}

	if len(res.Dropped) != 1 {
		t.Fatalf("advantage should drop 1 die, got %d", len(res.Dropped))
	}

	if res.Kept[0] < res.Dropped[0] {
		t.Errorf("kept die (%d) is lower than dropped die (%d) in advantage", res.Kept[0], res.Dropped[0])
	}
}

func TestRollModifier(t *testing.T) {
	expr := &parser.DiceExpr{
		Raw: "1d1+5",
	}

	res, err := Roll(expr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Total != 6 {
		t.Errorf("expected total 6 (1 + 5), got %d", res.Total)
	}
}

func TestRollKeepDrop(t *testing.T) {
	expr := &parser.DiceExpr{
		Raw: "4d6kh3",
	}

	res, err := Roll(expr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res.RawRolls) != 4 {
		t.Fatalf("expected 4 raw rolls, got %d", len(res.RawRolls))
	}

	if len(res.Kept) != 3 {
		t.Fatalf("expected 3 kept rolls, got %d", len(res.Kept))
	}

	if len(res.Dropped) != 1 {
		t.Fatalf("expected 1 dropped roll, got %d", len(res.Dropped))
	}

	// Verify kept are >= dropped
	droppedVal := res.Dropped[0]
	for _, k := range res.Kept {
		if k < droppedVal {
			t.Errorf("kept value %d is less than dropped value %d", k, droppedVal)
		}
	}
}
