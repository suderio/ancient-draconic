package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
)

func TestExecuteDamageWithDefenses(t *testing.T) {
	state := engine.NewGameState()
	state.Entities["elara"] = &engine.Entity{
		ID:        "elara",
		Name:      "Elara",
		Resources: map[string]int{"hp": 30, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10},
		Statuses:  make(map[string]string),
	}

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})

	tests := []struct {
		name     string
		res      []string
		imm      []string
		vul      []string
		expected int // Final damage amount
	}{
		{"Fire Resistance (10 -> 5)", []string{"fire"}, nil, nil, 5},
		{"Poison Immunity (10 -> 0)", nil, []string{"poison"}, nil, 0},
		{"Cold Vulnerability (10 -> 20)", nil, nil, []string{"cold"}, 20},
		{"Normal Damage (10 -> 10)", nil, nil, nil, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state.Entities["elara"].Resistances = tt.res
			state.Entities["elara"].Immunities = tt.imm
			state.Entities["elara"].Vulnerabilities = tt.vul
			state.Entities["elara"].Spent["hp"] = 0

			params := map[string]any{
				"dice": "10",
				"type": func() string {
					if len(tt.res) > 0 {
						return tt.res[0]
					}
					if len(tt.imm) > 0 {
						return tt.imm[0]
					}
					if len(tt.vul) > 0 {
						return tt.vul[0]
					}
					return "slashing"
				}(),
			}

			events, err := ExecuteGenericCommand("damage", "GM", []string{"elara"}, params, "", state, loader, testReg(loader))
			assert.NoError(t, err)

			for _, e := range events {
				e.Apply(state)
			}
			assert.Equal(t, tt.expected, state.Entities["elara"].Spent["hp"])
		})
	}
}

func TestExecuteDamageDefaultWeapon(t *testing.T) {
	state := engine.NewGameState()
	state.Entities["goblin"] = &engine.Entity{
		ID:        "goblin",
		Name:      "Goblin",
		Resources: map[string]int{"hp": 7, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10},
		Statuses:  make(map[string]string),
	}

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})
	params := map[string]any{
		"dice": "5",
		"type": "piercing",
	}

	events, err := ExecuteGenericCommand("damage", "GM", []string{"goblin"}, params, "", state, loader, testReg(loader))
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 5, state.Entities["goblin"].Spent["hp"])
}

func TestExecuteDamageMultipleRolls(t *testing.T) {
	state := engine.NewGameState()
	state.Entities["orc"] = &engine.Entity{
		ID:        "orc",
		Name:      "Orc",
		Resources: map[string]int{"hp": 50, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10},
		Statuses:  make(map[string]string),
	}

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})

	// Damage 1: 5
	params1 := map[string]any{"dice": "5", "type": "slashing"}
	evts1, _ := ExecuteGenericCommand("damage", "GM", []string{"orc"}, params1, "", state, loader, testReg(loader))
	for _, e := range evts1 {
		e.Apply(state)
	}

	// Damage 2: 20 (vulnerable to fire)
	state.Entities["orc"].Vulnerabilities = []string{"fire"}
	params2 := map[string]any{"dice": "10", "type": "fire"}
	evts2, _ := ExecuteGenericCommand("damage", "GM", []string{"orc"}, params2, "", state, loader, testReg(loader))
	for _, e := range evts2 {
		e.Apply(state)
	}

	assert.Equal(t, 25, state.Entities["orc"].Spent["hp"])
}
