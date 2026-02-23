package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

func TestAdjudicationFlow(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Grog"] = &engine.Entity{ID: "Grog", Name: "Grog", HP: 100, MaxHP: 100, ActionsRemaining: 1, AttacksRemaining: 0}
	state.Entities["Goblin"] = &engine.Entity{ID: "Goblin", Name: "Goblin", HP: 7, MaxHP: 7}
	state.Initiatives = map[string]int{"Grog": 20, "Goblin": 10}
	state.TurnOrder = []string{"Grog", "Goblin"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Initiate grapple (should trigger adjudication)
	cmd := &parser.GrappleCmd{Target: "Goblin"}
	events, err := ExecuteGrapple(cmd, state, loader)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.IsType(t, &engine.AdjudicationStartedEvent{}, events[0])

	// Apply event
	err = events[0].Apply(state)
	assert.NoError(t, err)
	assert.NotNil(t, state.PendingAdjudication)
	assert.True(t, state.IsFrozen())

	// 2. Allow adjudication
	allowCmd := &parser.AllowCmd{}
	events, err = ExecuteAllow(allowCmd, state)
	assert.NoError(t, err)
	assert.Len(t, events, 1)

	// Apply event (marks Approved, does not clear yet)
	err = events[0].Apply(state)
	assert.NoError(t, err)
	assert.NotNil(t, state.PendingAdjudication)
	assert.True(t, state.PendingAdjudication.Approved)
	assert.False(t, state.IsFrozen()) // Approved adjudication does NOT freeze

	// 3. Resume grapple (re-execution logic would be in Session, but we test the command's second stage)
	events, err = ExecuteGrapple(cmd, state, loader)
	assert.NoError(t, err)
	assert.Len(t, events, 2) // GrappleTaken + AskIssued
	assert.IsType(t, &engine.GrappleTakenEvent{}, events[0])
	assert.IsType(t, &engine.AskIssuedEvent{}, events[1])
}

func TestActionEconomy(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Paulo"] = &engine.Entity{ID: "Paulo", Name: "Paulo", HP: 20, MaxHP: 20, ActionsRemaining: 1, AttacksRemaining: 0}
	state.Initiatives = map[string]int{"Paulo": 15}
	state.TurnOrder = []string{"Paulo"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Take Dodge (uses action)
	dodgeCmd := &parser.DodgeCmd{}
	events, err := ExecuteDodge(dodgeCmd, state)
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)
	assert.Contains(t, state.Entities["Paulo"].Conditions, "Dodging")

	// 2. Try another action (should fail)
	_, err = ExecuteDodge(dodgeCmd, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions remaining")

	// 3. Verify Checks are free
	state.Entities["Paulo"].ActionsRemaining = 1
	checkCmd := &parser.CheckCmd{Actor: &parser.ActorExpr{Name: "Paulo"}, Check: []string{"Athletics"}}
	events, err = ExecuteCheck(checkCmd, state, loader)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 1, state.Entities["Paulo"].ActionsRemaining)

	// 4. Test Attack consumption
	state.Entities["Paulo"].ActionsRemaining = 1
	state.Entities["Paulo"].AttacksRemaining = 0

	attackCmd := &parser.AttackCmd{Weapon: "longsword", Targets: []string{"Goblin"}}

	events, err = ExecuteAttack(attackCmd, state, loader)
	// Even if it fails due to missing targets/AC, we can check the Apply logic of AttackResolvedEvent

	evt := &engine.AttackResolvedEvent{Attacker: "Paulo", Targets: []string{"Goblin"}}
	evt.Apply(state)
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)

	// 5. Test Action logic
	state.Entities["Paulo"].ActionsRemaining = 1
	actionCmd := &parser.ActionCmd{Action: "dash"}
	events, err = ExecuteAction(actionCmd, state, loader)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)
}

func TestHelpAction(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Paulo"] = &engine.Entity{ID: "Paulo", Name: "Paulo", ActionsRemaining: 1}
	state.Entities["Elara"] = &engine.Entity{ID: "Elara", Name: "Elara", HP: 10, MaxHP: 10}
	state.Entities["Orc"] = &engine.Entity{ID: "Orc", Name: "Orc", HP: 15, MaxHP: 15}
	state.Initiatives = map[string]int{"Paulo": 20, "Elara": 15, "Orc": 10}
	state.TurnOrder = []string{"Paulo", "Elara", "Orc"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Paulo helps Elara with a check
	helpCmd := &parser.HelpActionCmd{Type: "check", Target: "Elara"}
	// First call triggers adjudication
	events, err := ExecuteHelpAction(helpCmd, state)
	assert.NoError(t, err)
	assert.IsType(t, &engine.AdjudicationStartedEvent{}, events[0])
	events[0].Apply(state)

	// GM allows
	state.PendingAdjudication.Approved = true
	events, err = ExecuteHelpAction(helpCmd, state)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	assert.Contains(t, state.Entities["Elara"].Conditions, "HelpedCheck:Paulo")
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)

	// 2. Elara makes a check, gets advantage, and condition is removed
	checkCmd := &parser.CheckCmd{Actor: &parser.ActorExpr{Name: "Elara"}, Check: []string{"Athletics"}}
	events, err = ExecuteCheck(checkCmd, state, loader)
	assert.NoError(t, err)

	foundRemoved := false
	for _, e := range events {
		if _, ok := e.(*engine.ConditionRemovedEvent); ok {
			foundRemoved = true
		}
		e.Apply(state)
	}
	assert.True(t, foundRemoved)
	assert.NotContains(t, state.Entities["Elara"].Conditions, "HelpedCheck:Paulo")

	// 3. Help for attack
	state.Entities["Paulo"].ActionsRemaining = 1
	helpAttackCmd := &parser.HelpActionCmd{Type: "attack", Target: "Orc"}
	// Mock approved adjudication
	state.PendingAdjudication = &engine.PendingAdjudicationState{Approved: true}
	events, err = ExecuteHelpAction(helpAttackCmd, state)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Contains(t, state.Entities["Orc"].Conditions, "HelpedAttack:Paulo")

	// 4. Expiration
	// Change turn back to Paulo
	turnEvent := &engine.TurnChangedEvent{ActorID: "Paulo"}
	turnEvent.Apply(state)
	assert.NotContains(t, state.Entities["Orc"].Conditions, "HelpedAttack:Paulo")
}

func TestDynamicGrappleDC(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	// Thorne has 16 Str (+3) and 2 Prof Bonus -> DC should be 8 + 3 + 2 = 13
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1}
	state.Entities["Goblin"] = &engine.Entity{ID: "Goblin", Name: "Goblin"}
	state.Initiatives = map[string]int{"thorne": 20, "Goblin": 10}
	state.TurnOrder = []string{"thorne", "Goblin"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})
	cmd := &parser.GrappleCmd{Target: "Goblin"}

	// Mock approved adjudication
	state.PendingAdjudication = &engine.PendingAdjudicationState{Approved: true}

	events, err := ExecuteGrapple(cmd, state, loader)
	assert.NoError(t, err)

	foundAsk := false
	for _, e := range events {
		if ask, ok := e.(*engine.AskIssuedEvent); ok {
			foundAsk = true
			assert.Equal(t, 13, ask.DC, "DC should be 8 + 3 (Str) + 2 (Prof) = 13")
		}
	}
	assert.True(t, foundAsk)
}

func TestDynamicEscapeDC(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	// Thorne has 16 Str (+3) and 2 Prof Bonus -> DC should be 8 + 3 + 2 = 13
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1}
	state.Entities["goblin"] = &engine.Entity{ID: "goblin", Name: "Goblin", ActionsRemaining: 1, Conditions: []string{"grappledby:thorne"}}
	state.Initiatives = map[string]int{"thorne": 20, "goblin": 10}
	state.TurnOrder = []string{"thorne", "goblin"}
	state.CurrentTurn = 1

	loader := data.NewLoader([]string{"../../data"})
	cmd := &parser.ActionCmd{Action: "escape"}

	events, err := ExecuteAction(cmd, state, loader)
	assert.NoError(t, err)

	foundAsk := false
	for _, e := range events {
		if ask, ok := e.(*engine.AskIssuedEvent); ok {
			foundAsk = true
			assert.Equal(t, 13, ask.DC, "Escape DC should match Thorne's Grapple DC (13)")
			assert.Equal(t, "grappledby:thorne", ask.Succeeds.RemoveCondition)
		}
	}
	assert.True(t, foundAsk)
}

func TestShoveMechanic(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	loader := data.NewLoader([]string{"../../data"})

	// Thorne (Medium) shoves a Goblin (Small) -> OK
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1, Type: "Character"}
	state.Entities["goblin"] = &engine.Entity{ID: "goblin", Name: "Goblin", Type: "Monster"}
	state.Initiatives = map[string]int{"thorne": 20, "goblin": 10}
	state.TurnOrder = []string{"thorne", "goblin"}
	state.CurrentTurn = 0

	cmd := &parser.ActionCmd{Action: "shove", Target: "goblin"}
	events, err := ExecuteShove(cmd, state, loader)
	assert.NoError(t, err)
	assert.Equal(t, 13, events[2].(*engine.AskIssuedEvent).DC) // 8 + 3 (Str) + 2 (Prof)

	// Thorne (Medium) shoves a T-Rex (Huge) -> FAIL (Too large)
	state.Entities["t-rex"] = &engine.Entity{ID: "t-rex", Name: "T-Rex", Type: "Monster"}
	state.Initiatives["t-rex"] = 5 // Avoid freezing the game
	cmdHuge := &parser.ActionCmd{Action: "shove", Target: "t-rex"}
	_, err = ExecuteShove(cmdHuge, state, loader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestDisengageLogic(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	loader := data.NewLoader([]string{"../../data"})

	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1}
	state.Initiatives = map[string]int{"thorne": 20}
	state.TurnOrder = []string{"thorne"}
	state.CurrentTurn = 0

	// 1. Take Disengage
	cmd := &parser.ActionCmd{Action: "disengage"}
	events, err := ExecuteAction(cmd, state, loader)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Contains(t, state.Entities["thorne"].Conditions, "Disengaged")

	// 2. End Turn -> Disengaged should be cleared
	turnCmd := &parser.TurnCmd{}
	events, err = ExecuteTurn(turnCmd, state, loader)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.NotContains(t, state.Entities["thorne"].Conditions, "Disengaged")
}
func TestSavingThrowProficiency(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	// Elara: Dex 16 (+3), Prof 2. Saving Throw Dex should be +5.
	state.Entities["elara"] = &engine.Entity{ID: "elara", Name: "Elara"}
	state.Initiatives = map[string]int{"elara": 20}
	state.TurnOrder = []string{"elara"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Regular Check (No proficiency)
	cmdCheck := &parser.CheckCmd{Actor: &parser.ActorExpr{Name: "elara"}, Check: []string{"dex"}}
	events, err := ExecuteCheck(cmdCheck, state, loader)
	assert.NoError(t, err)
	foundCheck := false
	for _, e := range events {
		if dr, ok := e.(*engine.DiceRolledEvent); ok {
			foundCheck = true
			assert.Equal(t, 3, dr.Modifier, "Regular Dex check should only have +3 modifier")
		}
	}
	assert.True(t, foundCheck)

	// 2. Saving Throw (With proficiency)
	cmdSave := &parser.CheckCmd{Actor: &parser.ActorExpr{Name: "elara"}, Check: []string{"dex", "save"}}
	events, err = ExecuteCheck(cmdSave, state, loader)
	assert.NoError(t, err)
	foundSave := false
	for _, e := range events {
		if dr, ok := e.(*engine.DiceRolledEvent); ok {
			foundSave = true
			assert.Equal(t, 5, dr.Modifier, "Dex saving throw should have +5 modifier (3 + 2)")
		}
	}
	assert.True(t, foundSave)
}

func TestTwoWeaponFighting(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	// Thorne: Str 16 (+3). Longsword: 1d8+3.
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1, BonusActionsRemaining: 1}
	state.Entities["goblin"] = &engine.Entity{ID: "goblin", Name: "Goblin"}
	state.Initiatives = map[string]int{"thorne": 20, "goblin": 10}
	state.TurnOrder = []string{"thorne", "goblin"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Off-hand Attack should fail if no prior attack this turn
	cmdFailNoAttack := &parser.AttackCmd{OffHand: true, Weapon: "dagger", Targets: []string{"goblin"}, Dice: &parser.DiceExpr{Raw: "1d20+5"}}
	_, err := ExecuteAttack(cmdFailNoAttack, state, loader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must take the Attack action before")

	// 2. Main Attack (enables off-hand)
	mainCmd := &parser.AttackCmd{Weapon: "longsword", Targets: []string{"goblin"}, Dice: &parser.DiceExpr{Raw: "1d1+20"}}
	mainEvents, err := ExecuteAttack(mainCmd, state, loader)
	assert.NoError(t, err)
	for _, e := range mainEvents {
		e.Apply(state)
	}
	assert.True(t, state.Entities["thorne"].HasAttackedThisTurn)
	assert.Equal(t, "Longsword", state.Entities["thorne"].LastAttackedWithWeapon)

	// 3. Off-hand Attack should fail if same weapon
	cmdFailSameWeapon := &parser.AttackCmd{OffHand: true, Weapon: "longsword", Targets: []string{"goblin"}}
	_, err = ExecuteAttack(cmdFailSameWeapon, state, loader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must use a different weapon")

	// 4. Success with different weapon (guaranteed hit)
	cmd := &parser.AttackCmd{OffHand: true, Weapon: "dagger", Targets: []string{"goblin"}, Dice: &parser.DiceExpr{Raw: "1d1+20"}}
	events, err := ExecuteAttack(cmd, state, loader)
	if !assert.NoError(t, err) {
		return
	}

	for _, e := range events {
		e.Apply(state)
	}

	assert.Equal(t, 0, state.Entities["thorne"].BonusActionsRemaining)
	assert.NotNil(t, state.PendingDamage)
	assert.True(t, state.PendingDamage.IsOffHand)

	// 5. Damage Resolution (Stripping modifier)
	dmgCmd := &parser.DamageCmd{
		Rolls: []*parser.DamageRollExpr{
			{Dice: &parser.DiceExpr{Raw: "1d4+3"}, Type: "piercing"},
		},
	}
	dmgEvents, err := ExecuteDamage(dmgCmd, state, loader)
	assert.NoError(t, err)

	foundDice := false
	for _, e := range dmgEvents {
		if dr, ok := e.(*engine.DiceRolledEvent); ok {
			foundDice = true
			// If we used a simulated dagger (1d4+3), it should be stripped to 1d4.
			// ExecuteDamage strips positive modifiers for IsOffHand.
			assert.Equal(t, 0, dr.Modifier, "Off-hand attack damage should have no positive modifier")
			assert.Equal(t, 1, len(dr.RawRolls), "Should roll 1d4")
		}
	}
	assert.True(t, foundDice)
}

func TestOpportunityAttack(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ReactionsRemaining: 1}
	state.Entities["elara"] = &engine.Entity{ID: "elara", Name: "Elara"}
	state.Initiatives = map[string]int{"thorne": 20, "elara": 10}
	state.TurnOrder = []string{"thorne", "elara"}
	state.CurrentTurn = 1 // Elara's turn

	loader := data.NewLoader([]string{"../../data"})

	// 1. Thorne takes reaction attack (Opportunity Attack)
	cmd := &parser.AttackCmd{Opportunity: true, Actor: &parser.ActorExpr{Name: "thorne"}, Weapon: "longsword", Targets: []string{"elara"}}

	// Should trigger adjudication
	events, err := ExecuteAttack(cmd, state, loader)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.IsType(t, &engine.AdjudicationStartedEvent{}, events[0])

	events[0].Apply(state)
	assert.NotNil(t, state.PendingAdjudication)
	assert.Contains(t, state.PendingAdjudication.OriginalCommand, "opportunity attack")

	// GM Allows
	state.PendingAdjudication.Approved = true
	events, err = ExecuteAttack(cmd, state, loader)
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}

	assert.Equal(t, 0, state.Entities["thorne"].ReactionsRemaining)
}
