package command

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
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

	// 1. Dragon uses Fire Breath
	attackCmd := &parser.AttackCmd{Weapon: "Fire Breath", Targets: []string{"player"}}
	events, err := ExecuteAttack(attackCmd, state, loader)
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
	_, err = ExecuteAttack(attackCmd, state, loader)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cooling down")

	// 3. End Dragon's turn (rotate to Player)
	turnCmd := &parser.TurnCmd{}
	events, err = ExecuteTurn(turnCmd, state, loader)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 1, state.CurrentTurn)
	assert.Equal(t, "player", state.TurnOrder[state.CurrentTurn])

	// 4. End Player's turn (rotate to Dragon)
	// This is where recharge roll happens
	events, err = ExecuteTurn(turnCmd, state, loader)
	assert.NoError(t, err)

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
		_, err = ExecuteAttack(attackCmd, state, loader)
		assert.NoError(t, err)
	} else {
		// Not recharged! Attack should still fail
		state.Entities["dragon"].ActionsRemaining = 1
		_, err = ExecuteAttack(attackCmd, state, loader)
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "cooling down")
	}
}
