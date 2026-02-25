package command

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/rules"
)

func TestMonsterRecharge(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["dragon"] = &engine.Entity{
		ID:        "dragon",
		Name:      "dragon",
		Resources: map[string]int{"hp": 20, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10},
		Statuses:  make(map[string]string),
	}
	state.Entities["player"] = &engine.Entity{
		ID:        "player",
		Name:      "player",
		Resources: map[string]int{"hp": 20, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10, "ac": 12},
		Statuses:  make(map[string]string),
	}
	state.Metadata["initiatives"] = map[string]int{"dragon": 20, "player": 10}
	state.TurnOrder = []string{"dragon", "player"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})
	manifest, _ := loader.LoadManifest()
	reg, _ := rules.NewRegistry(manifest, func(s string) int { return 10 }, nil)

	// 1. Dragon uses Fire Breath
	params := map[string]any{"weapon": "Fire Breath"}
	events, err := ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
	assert.NoError(t, err)

	spentFound := false
	resolvedName := ""
	for _, e := range events {
		if se, ok := e.(*engine.AbilitySpentEvent); ok {
			spentFound = true
			resolvedName = se.ActionName
		}
		e.Apply(state)
	}
	assert.True(t, spentFound, "AbilitySpentEvent should be emitted")
	spentRecharges, _ := state.Metadata["spent_recharges"].(map[string][]string)
	assert.Contains(t, spentRecharges["dragon"], resolvedName)

	// 2. Try to use it again (should fail)
	state.Entities["dragon"].Spent["actions"] = 0 // Reset action to test recharge block specifically
	_, err = ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cooling down")

	// 3. End Dragon's turn
	events, err = ExecuteGenericCommand("turn", "dragon", []string{"dragon"}, nil, "", state, loader, testReg(loader))
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	// 4. End Player's turn (rotate to Dragon) -> Recharge attempt
	events, err = ExecuteGenericCommand("turn", "player", []string{"player"}, nil, "", state, loader, testReg(loader))
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	// 5. Final check of recharge state
	spentRechargesMap, _ := state.Metadata["spent_recharges"].(map[string][]string)
	if len(spentRechargesMap["dragon"]) == 0 {
		// Recharged!
		state.Entities["dragon"].Spent["actions"] = 0
		_, err = ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
		assert.NoError(t, err)
	} else {
		// Not recharged!
		state.Entities["dragon"].Spent["actions"] = 0
		_, err = ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
		assert.Error(t, err)
	}
}
