package command_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suderio/ancient-draconic/internal/command"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

func testReg() *rules.Registry {
	reg, _ := rules.NewRegistry(func(s string) int { return 10 })
	return reg
}

func TestExecuteDamageWithDefenses(t *testing.T) {
	// Setup a temporary data directory with elara.yaml
	tmpDir, err := os.MkdirTemp("", "dndsl-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	charDir := filepath.Join(tmpDir, "characters")
	os.MkdirAll(charDir, 0755)

	elaraYaml := `
name: Elara Shadowstep
hit_points: 30
defenses:
  - resistances: [fire]
    immunities: [poison]
    vulnerabilities: [cold]
`
	err = os.WriteFile(filepath.Join(charDir, "elara.yaml"), []byte(elaraYaml), 0644)
	if err != nil {
		t.Fatal(err)
	}

	loader := data.NewLoader([]string{tmpDir})

	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["elara"] = &engine.Entity{
		ID:       "elara",
		Name:     "Elara Shadowstep",
		HP:       30,
		MaxHP:    30,
		Category: "Character",
	}
	state.TurnOrder = []string{"gm", "elara"}
	state.CurrentTurn = 0
	state.Initiatives["gm"] = 10
	state.Initiatives["elara"] = 20

	// Mock a pending attack from GM to Elara
	state.PendingDamage = &engine.PendingDamageState{
		Attacker: "gm",
		Targets:  []string{"elara"},
		Weapon:   "Claws",
		HitStatus: map[string]bool{
			"elara": true,
		},
	}

	tests := []struct {
		name     string
		dice     string
		dmgType  string
		expected int // Negative amount for HP change
	}{
		{
			name:     "Fire Resistance (10 -> 5)",
			dice:     "10d1", // Using static dice for predictable test
			dmgType:  "fire",
			expected: -5,
		},
		{
			name:     "Poison Immunity (10 -> 0)",
			dice:     "10d1",
			dmgType:  "poison",
			expected: 0,
		},
		{
			name:     "Cold Vulnerability (10 -> 20)",
			dice:     "10d1",
			dmgType:  "cold",
			expected: -20,
		},
		{
			name:     "Normal Damage (10 -> 10)",
			dice:     "10d1",
			dmgType:  "slashing",
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdObj := &parser.DamageCmd{
				Rolls: []*parser.DamageRollExpr{
					{
						Dice: &parser.DiceExpr{Raw: tt.dice},
						Type: tt.dmgType,
					},
				},
			}

			events, err := command.ExecuteDamage(cmdObj, state, loader, testReg())
			if err != nil {
				t.Fatalf("ExecuteDamage failed: %v", err)
			}

			foundHPChange := false
			for _, ev := range events {
				if hpc, ok := ev.(*engine.HPChangedEvent); ok {
					if hpc.ActorID == "elara" {
						if hpc.Amount != tt.expected {
							t.Errorf("Expected HP change %d, got %d", tt.expected, hpc.Amount)
						}
						foundHPChange = true
					}
				}
			}

			if !foundHPChange {
				t.Errorf("HPChangedEvent not found in events")
			}
		})
	}
}

func TestExecuteDamageDefaultWeapon(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dndsl-test-default-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	charDir := filepath.Join(tmpDir, "characters")
	os.MkdirAll(charDir, 0755)

	// Elara has resistance to fire
	elaraYaml := `
name: Elara Shadowstep
hit_points: 30
defenses:
  - resistances: [fire]
`
	// Goblin has a fire weapon
	goblinYaml := `
name: Goblin
actions:
  - name: FireSword
    damage:
      - damage_dice: 10d1
        damage_type:
          index: fire
`
	os.WriteFile(filepath.Join(charDir, "elara.yaml"), []byte(elaraYaml), 0644)
	os.WriteFile(filepath.Join(charDir, "goblin.yaml"), []byte(goblinYaml), 0644)

	loader := data.NewLoader([]string{tmpDir})

	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["elara"] = &engine.Entity{ID: "elara", Name: "Elara", HP: 30, Category: "Character"}
	state.Entities["goblin"] = &engine.Entity{ID: "goblin", Name: "Goblin", HP: 10, Category: "Monster"}
	state.Initiatives["elara"] = 20
	state.Initiatives["goblin"] = 10
	state.TurnOrder = []string{"elara", "goblin"}
	state.CurrentTurn = 1 // Goblin's turn to deal damage

	state.PendingDamage = &engine.PendingDamageState{
		Attacker:  "goblin",
		Targets:   []string{"elara"},
		Weapon:    "FireSword",
		HitStatus: map[string]bool{"elara": true},
	}

	cmdObj := &parser.DamageCmd{
		Weapon: "FireSword",
	}

	events, err := command.ExecuteDamage(cmdObj, state, loader, testReg())
	if err != nil {
		t.Fatalf("ExecuteDamage failed: %v", err)
	}

	for _, ev := range events {
		if hpc, ok := ev.(*engine.HPChangedEvent); ok {
			if hpc.Amount != -5 { // 10 -> 5 due to fire resistance
				t.Errorf("Expected HP change -5, got %d", hpc.Amount)
			}
			return
		}
	}
	t.Errorf("HPChangedEvent not found")
}

func TestExecuteDamageMultipleRolls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dndsl-test-multi-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	charDir := filepath.Join(tmpDir, "characters")
	os.MkdirAll(charDir, 0755)

	elaraYaml := `
name: Elara Shadowstep
hit_points: 30
defenses:
  - resistances: [fire]
    vulnerabilities: [cold]
`
	os.WriteFile(filepath.Join(charDir, "elara.yaml"), []byte(elaraYaml), 0644)
	loader := data.NewLoader([]string{tmpDir})

	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["elara"] = &engine.Entity{ID: "elara", Name: "Elara", HP: 30, Category: "Character"}
	state.Initiatives["elara"] = 20
	state.Initiatives["gm"] = 10 // Need GM initiative if GM is the attacker
	state.TurnOrder = []string{"elara", "gm"}
	state.CurrentTurn = 1

	state.PendingDamage = &engine.PendingDamageState{
		Attacker:  "gm",
		Targets:   []string{"elara"},
		HitStatus: map[string]bool{"elara": true},
	}

	// 10 fire (resist -> 5) + 10 cold (vuln -> 20) = 25 total
	cmdObj := &parser.DamageCmd{
		Rolls: []*parser.DamageRollExpr{
			{Dice: &parser.DiceExpr{Raw: "10d1"}, Type: "fire"},
			{Dice: &parser.DiceExpr{Raw: "10d1"}, Type: "cold"},
		},
	}

	events, err := command.ExecuteDamage(cmdObj, state, loader, testReg())
	if err != nil {
		t.Fatalf("ExecuteDamage failed: %v", err)
	}

	for _, ev := range events {
		if hpc, ok := ev.(*engine.HPChangedEvent); ok {
			if hpc.Amount != -25 {
				t.Errorf("Expected HP change -25, got %d", hpc.Amount)
			}
			return
		}
	}
	t.Errorf("HPChangedEvent not found")
}
