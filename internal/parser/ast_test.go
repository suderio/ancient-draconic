package parser_test

import (
	"testing"

	"github.com/suderio/ancient-draconic/internal/parser"
)

func TestParseEncounterStart(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "encounter by: GM start with: Goblin and: Paulo")
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

func TestParseAddCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "add by: GM Dragon and: Mage")
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

func TestParseGenericCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "attack by: Goblin with: Scimitar to: Elara and: Paulo dice: 1d20+10")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Generic == nil {
		t.Fatalf("Expected GenericCmd, got nil")
	}

	if cmd.Generic.Name != "attack" {
		t.Errorf("Expected Command attack, got %s", cmd.Generic.Name)
	}

	if cmd.Generic.Actor == nil || cmd.Generic.Actor.Name != "Goblin" {
		t.Errorf("Expected Actor Goblin")
	}

	if len(cmd.Generic.Args) != 3 {
		t.Fatalf("Expected 3 generic args, got %d", len(cmd.Generic.Args))
	}

	if cmd.Generic.Args[0].Key != "with" || cmd.Generic.Args[0].Values[0] != "Scimitar" {
		t.Errorf("Expected Arg[0] 'with: Scimitar', got %v : %v", cmd.Generic.Args[0].Key, cmd.Generic.Args[0].Values)
	}

	if cmd.Generic.Args[1].Key != "to" || len(cmd.Generic.Args[1].Values) != 2 || cmd.Generic.Args[1].Values[0] != "Elara" || cmd.Generic.Args[1].Values[1] != "Paulo" {
		t.Errorf("Expected Arg[1] 'to: Elara and Paulo', got %v : %v", cmd.Generic.Args[1].Key, cmd.Generic.Args[1].Values)
	}

	if cmd.Generic.Args[2].Key != "dice" || cmd.Generic.Args[2].Values[0] != "1d20+10" {
		t.Errorf("Expected Arg[2] 'dice: 1d20+10', got %v : %v", cmd.Generic.Args[2].Key, cmd.Generic.Args[2].Values)
	}
}

func TestParseHintCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "hint")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Hint == nil {
		t.Fatalf("Expected HintCmd, got nil")
	}
}
