package data

import (
	"testing"
)

func TestLoaderEmbeddedFallback(t *testing.T) {
	// Initialize loader with NO external directories
	l := NewLoader(nil)

	// Try to load a known baseline monster (Aboleth is common in SRD)
	monster, err := l.LoadMonster("Aboleth")
	if err != nil {
		t.Fatalf("Failed to load embedded monster: %v", err)
	}

	if monster.Name != "Aboleth" {
		t.Errorf("Expected Aboleth, got %s", monster.Name)
	}

	if monster.HitPoints == 0 {
		t.Error("Expected non-zero hit points for Aboleth")
	}

	// Try a character (Paulo is one of our test characters usually)
	char, err := l.LoadCharacter("Paulo")
	if err == nil {
		t.Logf("Found character %s", char.Name)
	}
}
