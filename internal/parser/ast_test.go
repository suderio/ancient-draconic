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

func TestParseAttackCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "attack :by Goblin :with Scimitar :to Elara :and Paulo :dice 1d20+10")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Attack == nil {
		t.Fatalf("Expected AttackCmd, got nil")
	}

	if cmd.Attack.Actor == nil || cmd.Attack.Actor.Name != "Goblin" {
		t.Errorf("Expected Actor Goblin")
	}

	if cmd.Attack.Weapon != "Scimitar" {
		t.Errorf("Expected Weapon Scimitar, got %s", cmd.Attack.Weapon)
	}

	if len(cmd.Attack.Targets) != 2 || cmd.Attack.Targets[0] != "Elara" || cmd.Attack.Targets[1] != "Paulo" {
		t.Errorf("Unexpected Targets: %v", cmd.Attack.Targets)
	}

	if cmd.Attack.Dice == nil || cmd.Attack.Dice.Raw != "1d20+10" {
		t.Errorf("Expected Dice macro 1d20+10")
	}
}

func TestParseDamageCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "damage :by Goblin :with Scimitar :dice 2d6+2")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Damage == nil {
		t.Fatalf("Expected DamageCmd, got nil")
	}

	if cmd.Damage.Actor == nil || cmd.Damage.Actor.Name != "Goblin" {
		t.Errorf("Expected Actor Goblin")
	}

	if cmd.Damage.Weapon != "Scimitar" {
		t.Errorf("Expected Weapon Scimitar, got %s", cmd.Damage.Weapon)
	}

	if len(cmd.Damage.Rolls) != 1 || cmd.Damage.Rolls[0].Dice == nil || cmd.Damage.Rolls[0].Dice.Raw != "2d6+2" {
		t.Errorf("Expected Dice macro 2d6+2")
	}
}

func TestParseTurnCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "turn :by Goblin")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Turn == nil {
		t.Fatalf("Expected TurnCmd, got nil")
	}

	if cmd.Turn.Actor == nil || cmd.Turn.Actor.Name != "Goblin" {
		t.Errorf("Expected Actor Goblin")
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

func TestParseAskCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "ask :by GM :check dex save :of goblin :and paulo :dc 15 :fails prone :succeeds damage 2d6")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Ask == nil {
		t.Fatalf("Expected AskCmd, got nil")
	}

	if cmd.Ask.DC != 15 {
		t.Errorf("Expected DC 15, got %d", cmd.Ask.DC)
	}

	if len(cmd.Ask.Check) != 2 || cmd.Ask.Check[0] != "dex" || cmd.Ask.Check[1] != "save" {
		t.Errorf("Expected Check to be ['dex', 'save'], got %v", cmd.Ask.Check)
	}

	if len(cmd.Ask.Targets) != 2 || cmd.Ask.Targets[0] != "goblin" || cmd.Ask.Targets[1] != "paulo" {
		t.Errorf("Expected Targets ['goblin', 'paulo'], got %v", cmd.Ask.Targets)
	}

	if cmd.Ask.Fails == nil || cmd.Ask.Fails.Condition != "prone" {
		t.Errorf("Expected Fails Condition 'prone', got %v", cmd.Ask.Fails)
	}

	if cmd.Ask.Succeeds == nil || cmd.Ask.Succeeds.DamageDice == nil || cmd.Ask.Succeeds.DamageDice.Raw != "2d6" {
		t.Errorf("Expected Succeeds Damage 2d6, got %v", cmd.Ask.Succeeds)
	}
}

func TestParseCheckCommand(t *testing.T) {
	p := parser.Build()

	cmd, err := p.ParseString("", "check :by goblin dex save")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if cmd.Check == nil {
		t.Fatalf("Expected CheckCmd, got nil")
	}

	if cmd.Check.Actor == nil || cmd.Check.Actor.Name != "goblin" {
		t.Errorf("Expected Actor 'goblin'")
	}

	if len(cmd.Check.Check) != 2 || cmd.Check.Check[0] != "dex" || cmd.Check.Check[1] != "save" {
		t.Errorf("Expected Check ['dex', 'save'], got %v", cmd.Check.Check)
	}
}
