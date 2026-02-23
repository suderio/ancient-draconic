package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

func testReg(loader *data.Loader) *rules.Registry {
	var m *data.CampaignManifest
	if loader != nil {
		m, _ = loader.LoadManifest()
	}
	reg, _ := rules.NewRegistry(m, func(s string) int { return 10 }, nil)
	return reg
}

func TestAdjudicationFlow(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Grog"] = &engine.Entity{ID: "Grog", Name: "Grog", HP: 100, MaxHP: 100, ActionsRemaining: 1, AttacksRemaining: 0, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Entities["Goblin"] = &engine.Entity{ID: "Goblin", Name: "Goblin", HP: 7, MaxHP: 7, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Initiatives = map[string]int{"Grog": 20, "Goblin": 10}
	state.TurnOrder = []string{"Grog", "Goblin"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Initiate grapple (should trigger adjudication)
	params := map[string]any{}
	events, err := ExecuteGenericCommand("grapple", "Grog", []string{"Goblin"}, params, "", state, loader, testReg(loader))
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
	events, err = ExecuteGenericCommand("grapple", "Grog", []string{"Goblin"}, params, "", state, loader, testReg(loader))
	assert.NoError(t, err)
	assert.Len(t, events, 2) // GrappleTaken + AskIssued
	assert.IsType(t, &engine.GrappleTakenEvent{}, events[0])
	assert.IsType(t, &engine.AskIssuedEvent{}, events[1])
}

func TestActionEconomy(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Paulo"] = &engine.Entity{ID: "Paulo", Name: "Paulo", HP: 20, MaxHP: 20, ActionsRemaining: 1, AttacksRemaining: 0, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Initiatives = map[string]int{"Paulo": 15}
	state.TurnOrder = []string{"Paulo"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Take Dodge (uses action)
	events, err := ExecuteGenericCommand("dodge", "Paulo", []string{"Paulo"}, nil, "", state, loader, testReg(loader))
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)
	assert.Contains(t, state.Entities["Paulo"].Conditions, "Dodging")

	// 2. Try another action (should fail)
	_, err = ExecuteGenericCommand("dodge", "Paulo", []string{"Paulo"}, nil, "", state, loader, testReg(loader))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions remaining")

	// 3. Verify Checks are free
	state.Entities["Paulo"].ActionsRemaining = 1
	// Define check command in registry
	checkReg, _ := rules.NewRegistry(&data.CampaignManifest{
		Commands: map[string]data.CommandDefinition{
			"check": {Name: "check", Steps: []data.CommandStep{{Name: "success", Formula: "true", Event: "CheckResolved"}}},
		},
	}, func(s string) int { return 10 }, nil)

	params := map[string]any{"check": "Athletics"}
	events, err = ExecuteGenericCommand("check", "Paulo", []string{"Paulo"}, params, "", state, loader, checkReg)
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 1, state.Entities["Paulo"].ActionsRemaining)

	// 4. Test Attack consumption
	state.Entities["Paulo"].ActionsRemaining = 1
	state.Entities["Paulo"].AttacksRemaining = 0

	attackReg, _ := rules.NewRegistry(&data.CampaignManifest{
		Commands: map[string]data.CommandDefinition{
			"attack": {Name: "attack", Steps: []data.CommandStep{{Name: "hit", Formula: "true", Event: "AttackResolved"}}},
		},
	}, func(s string) int { return 10 }, nil)

	params = map[string]any{"weapon": "longsword"}
	events, err = ExecuteGenericCommand("attack", "Paulo", []string{"Goblin"}, params, "", state, loader, attackReg)
	assert.NoError(t, err)
	// Even if it fails due to missing targets/AC, we can check the Apply logic of AttackResolvedEvent

	evt := &engine.AttackResolvedEvent{Attacker: "Paulo", Targets: []string{"Goblin"}}
	evt.Apply(state)
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)

	// 5. Test Action logic
	state.Entities["Paulo"].ActionsRemaining = 1
	actionCmd := &parser.ActionCmd{Action: "dash"}
	events, err = ExecuteAction(actionCmd, state, loader, testReg(loader))
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)
}

func TestHelpAction(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Paulo"] = &engine.Entity{ID: "Paulo", Name: "Paulo", ActionsRemaining: 1, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Entities["Elara"] = &engine.Entity{ID: "Elara", Name: "Elara", HP: 10, MaxHP: 10, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Entities["Orc"] = &engine.Entity{ID: "Orc", Name: "Orc", HP: 15, MaxHP: 15, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Initiatives = map[string]int{"Paulo": 20, "Elara": 15, "Orc": 10}
	state.TurnOrder = []string{"Paulo", "Elara", "Orc"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Paulo helps Elara with a check
	helpCmd := &parser.HelpActionCmd{Type: "check", Target: "Elara"}
	// First call triggers adjudication
	events, err := ExecuteHelpAction(helpCmd, state, testReg(loader))
	assert.NoError(t, err)
	assert.IsType(t, &engine.AdjudicationStartedEvent{}, events[0])
	events[0].Apply(state)

	// GM allows
	state.PendingAdjudication.Approved = true
	events, err = ExecuteHelpAction(helpCmd, state, testReg(loader))
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	assert.Contains(t, state.Entities["Elara"].Conditions, "HelpedCheck:Paulo")
	assert.Equal(t, 0, state.Entities["Paulo"].ActionsRemaining)

	// 2. Elara makes a check, gets advantage, and condition is removed
	reg, _ := rules.NewRegistry(nil, func(s string) int { return 10 }, nil) // Mock reg
	if m, err := loader.LoadManifest(); err == nil {
		reg, _ = rules.NewRegistry(m, func(s string) int { return 10 }, nil)
	}

	params := map[string]any{"check": "Athletics"}
	events, err = ExecuteGenericCommand("check", "Elara", []string{"Elara"}, params, "", state, loader, reg)
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
	events, err = ExecuteHelpAction(helpAttackCmd, state, testReg(loader))
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
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1, Stats: map[string]int{"str": 16, "prof_bonus": 2}}
	state.Entities["Goblin"] = &engine.Entity{ID: "Goblin", Name: "Goblin", Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Initiatives = map[string]int{"thorne": 20, "Goblin": 10}
	state.TurnOrder = []string{"thorne", "Goblin"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// Mock approved adjudication
	state.PendingAdjudication = &engine.PendingAdjudicationState{Approved: true}

	params := map[string]any{}
	events, err := ExecuteGenericCommand("grapple", "thorne", []string{"Goblin"}, params, "", state, loader, testReg(loader))
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
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ActionsRemaining: 1, Stats: map[string]int{"str": 16, "prof_bonus": 2}}
	state.Entities["goblin"] = &engine.Entity{ID: "goblin", Name: "Goblin", ActionsRemaining: 1, Conditions: []string{"grappledby:thorne"}, Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Initiatives = map[string]int{"thorne": 20, "goblin": 10}
	state.TurnOrder = []string{"thorne", "goblin"}
	state.CurrentTurn = 1

	loader := data.NewLoader([]string{"../../data"})
	cmd := &parser.ActionCmd{Action: "escape"}

	events, err := ExecuteAction(cmd, state, loader, testReg(loader))
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
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", Size: "medium", ActionsRemaining: 1, Category: "Character", Stats: map[string]int{"str": 16, "prof_bonus": 2}}
	state.Entities["goblin"] = &engine.Entity{ID: "goblin", Name: "Goblin", Size: "small", Category: "Monster"}
	state.Initiatives = map[string]int{"thorne": 20, "goblin": 10}
	state.TurnOrder = []string{"thorne", "goblin"}
	state.CurrentTurn = 0

	params := map[string]any{}
	events, err := ExecuteGenericCommand("shove", "thorne", []string{"goblin"}, params, "", state, loader, testReg(loader))
	assert.NoError(t, err)

	foundAsk := false
	for _, e := range events {
		if ask, ok := e.(*engine.AskIssuedEvent); ok {
			foundAsk = true
			assert.Equal(t, 13, ask.DC) // 8 + 3 (Str) + 2 (Prof)
		}
	}
	assert.True(t, foundAsk)

	// Thorne (Medium) shoves a T-Rex (Huge) -> FAIL (Too large)
	state.Entities["t-rex"] = &engine.Entity{ID: "t-rex", Name: "T-Rex", Size: "huge", Category: "Monster"}
	state.Initiatives["t-rex"] = 5 // Avoid freezing the game
	_, err = ExecuteGenericCommand("shove", "thorne", []string{"t-rex"}, params, "", state, loader, testReg(loader))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target is too large")
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
	events, err := ExecuteAction(cmd, state, loader, testReg(loader))
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}
	assert.Contains(t, state.Entities["thorne"].Conditions, "Disengaged")

	// 2. End Turn using generic engine
	events, err = ExecuteGenericCommand("turn", "thorne", []string{"thorne"}, nil, "", state, loader, testReg(loader))
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
	state.Entities["elara"] = &engine.Entity{
		ID:            "elara",
		Name:          "Elara",
		Stats:         map[string]int{"dex": 16, "prof_bonus": 2},
		Proficiencies: []string{"saving-throw-dex"},
	}
	state.Initiatives = map[string]int{"elara": 20}
	state.TurnOrder = []string{"elara"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../data"})

	// 1. Regular Check (No proficiency)
	reg, _ := rules.NewRegistry(nil, func(s string) int { return 10 }, nil)
	if m, err := loader.LoadManifest(); err == nil {
		reg, _ = rules.NewRegistry(m, func(s string) int { return 10 }, nil)
	}

	params := map[string]any{"check": "dex"}
	events, err := ExecuteGenericCommand("check", "elara", []string{"elara"}, params, "", state, loader, reg)
	assert.NoError(t, err)
	foundCheck := false
	for _, e := range events {
		if cr, ok := e.(*engine.CheckResolvedEvent); ok {
			foundCheck = true
			assert.Equal(t, 13, cr.Result, "Regular Dex check should have result 13 (10 base + 3 mod)")
		}
	}
	assert.True(t, foundCheck)

	// 2. Saving Throw (With proficiency)
	params = map[string]any{"check": "dex save"}
	events, err = ExecuteGenericCommand("check", "elara", []string{"elara"}, params, "", state, loader, reg)
	assert.NoError(t, err)
	foundSave := false
	for _, e := range events {
		if cr, ok := e.(*engine.CheckResolvedEvent); ok {
			foundSave = true
			assert.Equal(t, 15, cr.Result, "Dex saving throw should have result 15 (10 base + 3 mod + 2 prof)")
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
	reg, _ := rules.NewRegistry(nil, func(s string) int { return 10 }, nil)
	if m, err := loader.LoadManifest(); err == nil {
		reg, _ = rules.NewRegistry(m, func(s string) int { return 10 }, nil)
	}

	params := map[string]any{"weapon": "dagger", "offhand": true}
	_, err := ExecuteGenericCommand("attack", "thorne", []string{"goblin"}, params, "", state, loader, reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must take the Attack action before")

	// 2. Main Attack (enables off-hand)
	mainParams := map[string]any{"weapon": "longsword"}
	mainEvents, err := ExecuteGenericCommand("attack", "thorne", []string{"goblin"}, mainParams, "", state, loader, reg)
	assert.NoError(t, err)
	for _, e := range mainEvents {
		e.Apply(state)
	}
	assert.True(t, state.Entities["thorne"].HasAttackedThisTurn)
	assert.Equal(t, "Longsword", state.Entities["thorne"].LastAttackedWithWeapon)

	// 3. Off-hand Attack should fail if same weapon
	failParams := map[string]any{"weapon": "longsword", "offhand": true}
	_, err = ExecuteGenericCommand("attack", "thorne", []string{"goblin"}, failParams, "", state, loader, reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must use a different weapon")

	// 4. Success with different weapon (guaranteed hit)
	successParams := map[string]any{"weapon": "dagger", "offhand": true}
	events, err := ExecuteGenericCommand("attack", "thorne", []string{"goblin"}, successParams, "", state, loader, reg)
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}

	assert.Equal(t, 0, state.Entities["thorne"].BonusActionsRemaining)
	assert.NotNil(t, state.PendingDamage)
	assert.True(t, state.PendingDamage.IsOffHand)

	// 5. Damage Resolution (Stripping modifier)
	// 5. Damage Resolution using generic engine
	targets := []string{}
	for _, t := range state.PendingDamage.Targets {
		if state.PendingDamage.HitStatus[t] {
			targets = append(targets, t)
		}
	}
	params = map[string]any{
		"weapon":  state.PendingDamage.Weapon,
		"offhand": state.PendingDamage.IsOffHand,
		"dice":    "1d4+3",
		"type":    "piercing",
	}
	var dmgEvents []engine.Event
	dmgEvents, err = ExecuteGenericCommand("damage", state.PendingDamage.Attacker, targets, params, "", state, loader, testReg(loader))
	if err != nil {
		t.Fatalf("ExecuteGenericCommand failed: %v", err)
	}
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
	state.Entities["thorne"] = &engine.Entity{ID: "thorne", Name: "Thorne", ReactionsRemaining: 1, Stats: map[string]int{"str": 16, "prof_bonus": 2}}
	state.Entities["elara"] = &engine.Entity{ID: "elara", Name: "Elara", Stats: map[string]int{"str": 10, "dex": 10, "wis": 10}}
	state.Initiatives = map[string]int{"thorne": 20, "elara": 10}
	state.TurnOrder = []string{"thorne", "elara"}
	state.CurrentTurn = 1 // Elara's turn

	loader := data.NewLoader([]string{"../../data"})

	// 1. Thorne takes reaction attack (Opportunity Attack)
	// We'll use ExecuteGenericCommand directly to avoid import cycle
	params := map[string]any{
		"weapon":      "longsword",
		"opportunity": true,
	}

	// We need a registry with the command defined
	manifest := &data.CampaignManifest{
		Commands: map[string]data.CommandDefinition{
			"attack": {
				Name: "attack",
				Steps: []data.CommandStep{
					{Name: "check_opportunity", Formula: "action.opportunity && !manifest.approved ? 'adjudicate' : 'ok'"},
					{Name: "hit", Formula: "true", Event: "AttackResolved"},
				},
			},
		},
	}
	reg, _ := rules.NewRegistry(manifest, func(s string) int { return 10 }, nil)

	events, err := ExecuteGenericCommand("attack", "thorne", []string{"elara"}, params, "attack by: thorne opportunity", state, loader, reg)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.IsType(t, &engine.AdjudicationStartedEvent{}, events[0])

	events[0].Apply(state)
	assert.NotNil(t, state.PendingAdjudication)

	// GM Allows (simulate via manifest flag injection)
	state.PendingAdjudication.Approved = true
	events, err = ExecuteGenericCommand("attack", "thorne", []string{"elara"}, params, "attack by: thorne opportunity", state, loader, reg)
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}

	// The generic AttackResolved mapping currently doesn't decrement reactions,
	// but the engine's AttackResolved.Apply DOES if IsOpportunity is true.
	// We need to ensure mapManifestEvent sets IsOpportunity.
	assert.Equal(t, 0, state.Entities["thorne"].ReactionsRemaining)
}

type mockStore struct{}

func (m *mockStore) Append(e engine.Event) error   { return nil }
func (m *mockStore) Load() ([]engine.Event, error) { return nil, nil }
func (m *mockStore) Close() error                  { return nil }
