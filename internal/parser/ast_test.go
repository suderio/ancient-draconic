package parser_test

import (
	"dndsl/internal/parser"
	"testing"
)

func TestParseEncounterStart(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "encounter :by GM start :with Goblin :and Paulo")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Encounter == nil {
		t.Fatalf("Expected EncounterCmd, got nil")
	}

	if cmd.Encounter.Actor.Name != "GM" {
		t.Errorf("Expected GM actor, got %s", cmd.Encounter.Actor.Name)
	}

	if cmd.Encounter.Action != "start" {
		t.Errorf("Expected action start, got %s", cmd.Encounter.Action)
	}

	if len(cmd.Encounter.Targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(cmd.Encounter.Targets))
	}

	if cmd.Encounter.Targets[0] != "Goblin" || cmd.Encounter.Targets[1] != "Paulo" {
		t.Errorf("Unexpected targets: %v", cmd.Encounter.Targets)
	}
}

func TestParseEncounterEnd(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "encounter :by GM end")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Encounter == nil {
		t.Fatalf("Expected EncounterCmd, got nil")
	}

	if cmd.Encounter.Action != "end" {
		t.Errorf("Expected action end, got %s", cmd.Encounter.Action)
	}

	if len(cmd.Encounter.Targets) != 0 {
		t.Errorf("Expected 0 targets on end, got %d", len(cmd.Encounter.Targets))
	}
}

func TestParseAddCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "add :by GM Dragon :and Mage")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Add == nil {
		t.Fatalf("Expected AddCmd, got nil")
	}

	if len(cmd.Add.Targets) != 2 {
		t.Fatalf("Expected 2 targets, got %v", cmd.Add.Targets)
	}

	if cmd.Add.Targets[0] != "Dragon" || cmd.Add.Targets[1] != "Mage" {
		t.Errorf("Unexpected targets: %v", cmd.Add.Targets)
	}
}

func TestParseInitiativeCommand(t *testing.T) {
	p := parser.Build()

	t.Run("Auto Roll", func(t *testing.T) {
		cmd, err := p.ParseString("", "initiative :by Paulo")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if cmd.Initiative == nil {
			t.Fatalf("Expected InitiativeCmd, got nil")
		}

		if cmd.Initiative.Value != nil {
			t.Errorf("Expected nil value for auto roll, got %d", *cmd.Initiative.Value)
		}
	})

	t.Run("Manual Roll", func(t *testing.T) {
		cmd, err := p.ParseString("", "initiative :by Paulo 18")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if cmd.Initiative == nil {
			t.Fatalf("Expected InitiativeCmd, got nil")
		}

		if cmd.Initiative.Value == nil || *cmd.Initiative.Value != 18 {
			t.Errorf("Expected manual roll 18")
		}
	})
}
