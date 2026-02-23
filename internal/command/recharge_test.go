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
	state.Entities["dragon"] = &engine.Entity{ID: "dragon", Name: "dragon", ActionsRemaining: 1}
	state.Entities["player"] = &engine.Entity{ID: "player", Name: "player", HP: 20, MaxHP: 20}
	state.Initiatives = map[string]int{"dragon": 20, "player": 10}
	state.TurnOrder = []string{"dragon", "player"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})
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
	assert.Contains(t, state.SpentRecharges["dragon"], resolvedName)

	// 2. Try to use it again (should fail)
	state.Entities["dragon"].ActionsRemaining = 1 // Give another action to test recharge block
	_, err = ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cooling down")

	// 3. End Dragon's turn (rotate to Player)
	events, err = ExecuteGenericCommand("turn", "dragon", []string{"dragon"}, nil, "", state, loader, testReg(loader))
	if err != nil {
		t.Fatalf("ExecuteGenericCommand failed: %v", err)
	}
	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 1, state.CurrentTurn)
	assert.Equal(t, "player", state.TurnOrder[state.CurrentTurn])

	// 4. End Player's turn (rotate to Dragon)
	// This is where recharge roll happens
	events, err = ExecuteGenericCommand("turn", "player", []string{"player"}, nil, "", state, loader, testReg(loader))
	if err != nil {
		t.Fatalf("ExecuteGenericCommand failed: %v", err)
	}

	rollFound := false
	for _, e := range events {
		if _, ok := e.(*engine.RechargeRolledEvent); ok {
			rollFound = true
		}
		e.Apply(state)
	}
	assert.True(t, rollFound, "RechargeRolledEvent should be emitted")

	// Since the roll is random, we check both cases
	if len(state.SpentRecharges["dragon"]) == 0 {
		// Recharged! Try attack again
		state.Entities["dragon"].ActionsRemaining = 1
		_, err = ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
		assert.NoError(t, err)
	} else {
		// Not recharged! Attack should still fail
		state.Entities["dragon"].ActionsRemaining = 1
		_, err = ExecuteGenericCommand("attack", "dragon", []string{"player"}, params, "", state, loader, reg)
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "cooling down")
	}
}
