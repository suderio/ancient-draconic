package command

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/rules"
)

func testReg(loader *data.Loader) *rules.Registry {
	var m *data.CampaignManifest
	if loader != nil {
		m, _ = loader.LoadManifest()
	}
	reg, _ := rules.NewRegistry(m, func(s string) int {
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
		return 10
	}, nil)
	return reg
}

func TestActionEconomy(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Paulo"] = &engine.Entity{
		ID:        "Paulo",
		Name:      "Paulo",
		Resources: map[string]int{"hp": 20, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10, "dex": 10, "wis": 10, "prof_bonus": 2},
		Statuses:  make(map[string]string),
	}
	state.Metadata["initiatives"] = map[string]int{"Paulo": 15}
	state.TurnOrder = []string{"Paulo"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})

	// 1. Take Dodge (uses action)
	events, err := ExecuteGenericCommand("dodge", "Paulo", []string{"Paulo"}, nil, "", state, loader, testReg(loader))
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 1, state.Entities["Paulo"].Spent["actions"])
	assert.Contains(t, state.Entities["Paulo"].Conditions, "Dodging")

	// 2. Try another action (should fail)
	_, err = ExecuteGenericCommand("dodge", "Paulo", []string{"Paulo"}, nil, "", state, loader, testReg(loader))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions remaining")
}

func TestHelpAction(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Paulo"] = &engine.Entity{
		ID:        "Paulo",
		Name:      "Paulo",
		Resources: map[string]int{"actions": 1},
		Spent:     map[string]int{"actions": 0},
		Stats:     map[string]int{"str": 10, "dex": 10, "wis": 10, "prof_bonus": 2},
		Statuses:  make(map[string]string),
	}
	state.Entities["Elara"] = &engine.Entity{
		ID:        "Elara",
		Name:      "Elara",
		Resources: map[string]int{"hp": 10, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10, "dex": 10, "wis": 10, "prof_bonus": 2},
		Statuses:  make(map[string]string),
	}
	state.Entities["Orc"] = &engine.Entity{
		ID:        "Orc",
		Name:      "Orc",
		Resources: map[string]int{"hp": 15, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Stats:     map[string]int{"str": 10, "dex": 10, "wis": 10, "prof_bonus": 2},
		Statuses:  make(map[string]string),
	}
	state.Metadata["initiatives"] = map[string]int{"Paulo": 20, "Elara": 15, "Orc": 10}
	state.TurnOrder = []string{"Paulo", "Elara", "Orc"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})

	// 1. Paulo helps Elara with a check
	params := map[string]any{"type": "check", "target": "Elara"}
	// First call triggers adjudication
	events, err := ExecuteGenericCommand("help_action", "Paulo", []string{"Elara"}, params, "help check Elara", state, loader, testReg(loader))
	assert.NoError(t, err)
	assert.IsType(t, &engine.AdjudicationStartedEvent{}, events[0])
	events[0].Apply(state)

	// GM allows
	state.Metadata["pending_adjudication"] = map[string]any{"approved": true}
	events, err = ExecuteGenericCommand("help_action", "Paulo", []string{"Elara"}, params, "help check Elara", state, loader, testReg(loader))
	assert.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	assert.Contains(t, state.Entities["Elara"].Conditions, "HelpedCheck:Paulo")
	assert.Equal(t, 1, state.Entities["Paulo"].Spent["actions"])
}

func TestTwoWeaponFighting(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["thorne"] = &engine.Entity{
		ID:   "thorne",
		Name: "Thorne",
		Resources: map[string]int{
			"actions":       1,
			"bonus_actions": 1,
		},
		Spent: map[string]int{
			"hp":            0,
			"actions":       0,
			"bonus_actions": 0,
		},
		Stats:    map[string]int{"str": 16, "prof_bonus": 2},
		Statuses: make(map[string]string),
	}
	state.Entities["goblin"] = &engine.Entity{
		ID:        "goblin",
		Name:      "Goblin",
		Stats:     map[string]int{"str": 10, "ac": 12},
		Resources: map[string]int{"hp": 10, "actions": 1},
		Spent:     map[string]int{"hp": 0, "actions": 0},
		Statuses:  make(map[string]string),
	}
	state.Metadata["initiatives"] = map[string]int{"thorne": 20, "goblin": 10}
	state.TurnOrder = []string{"thorne", "goblin"}
	state.CurrentTurn = 0

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})
	reg := testReg(loader)

	// 1. Off-hand Attack should fail if no prior attack this turn
	params := map[string]any{"weapon": "dagger", "offhand": true}
	_, err := ExecuteGenericCommand("offhand_attack", "thorne", []string{"goblin"}, params, "", state, loader, reg)
	assert.Error(t, err)

	// 2. Main Attack (enables off-hand)
	mainParams := map[string]any{"weapon": "longsword"}
	mainEvents, err := ExecuteGenericCommand("attack", "thorne", []string{"goblin"}, mainParams, "", state, loader, reg)
	assert.NoError(t, err)
	for _, e := range mainEvents {
		e.Apply(state)
	}
	assert.Equal(t, "true", state.Entities["thorne"].Statuses["has_attacked_this_turn"])
	assert.Equal(t, "Longsword", state.Entities["thorne"].Statuses["last_attacked_with_weapon"])

	// 3. Success with different weapon
	successParams := map[string]any{"weapon": "dagger", "offhand": true}
	events, err := ExecuteGenericCommand("offhand_attack", "thorne", []string{"goblin"}, successParams, "", state, loader, reg)
	assert.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}

	assert.Equal(t, 1, state.Entities["thorne"].Spent["bonus_actions"])
	pendingDmg, ok := state.Metadata["pending_damage"].(map[string]any)
	if !ok {
		t.Fatalf("pending_damage not found in metadata")
	}
	assert.True(t, pendingDmg["is_off_hand"].(bool))

	// 4. Damage Resolution (Stripping modifier)
	targets := []string{"goblin"}
	attacker := "thorne"
	params = map[string]any{
		"weapon":  "dagger",
		"offhand": true,
		"dice":    "1d4+3",
		"type":    "piercing",
	}
	dmgEvents, err := ExecuteGenericCommand("damage", attacker, targets, params, "", state, loader, reg)
	assert.NoError(t, err)
	foundDice := false
	for _, e := range dmgEvents {
		if dr, ok := e.(*engine.DiceRolledEvent); ok {
			foundDice = true
			assert.Equal(t, 0, dr.Modifier)
		}
	}
	assert.True(t, foundDice)
}

func TestOpportunityAttack(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["thorne"] = &engine.Entity{
		ID:        "thorne",
		Name:      "Thorne",
		Resources: map[string]int{"actions": 1, "reactions": 1},
		Spent:     map[string]int{"actions": 0, "reactions": 0},
		Stats:     map[string]int{"str": 18, "ac": 15},
		Statuses:  make(map[string]string),
	}
	state.Entities["elara"] = &engine.Entity{
		ID:        "elara",
		Name:      "Elara",
		Resources: map[string]int{"actions": 1},
		Spent:     map[string]int{"actions": 0},
		Stats:     map[string]int{"str": 10, "ac": 15},
		Statuses:  make(map[string]string),
	}
	state.Metadata["initiatives"] = map[string]int{"thorne": 20, "elara": 10}
	state.TurnOrder = []string{"thorne", "elara"}
	state.CurrentTurn = 1

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})
	manifest := &data.CampaignManifest{
		Commands: map[string]data.CommandDefinition{
			"opportunity_attack": {
				Name: "opportunity_attack",
				Steps: []data.CommandStep{
					{Name: "check_opportunity", Formula: "pending_adjudication.approved ? 'ok' : 'adjudicate'"},
					{Name: "consume_reaction", Formula: "steps.check_opportunity == 'ok' ? {'actor_id': actor.id, 'type': 'spent', 'key': 'reactions', 'value': string(actor.spent.reactions + 1)} : {'type': 'skip', 'key': '', 'value': ''}", Event: "AttributeChanged"},
					{Name: "hit", Formula: "steps.check_opportunity == 'ok' ? {'key': 'pending_damage', 'value': {'attacker': actor.id, 'targets': [target.id], 'hit_status': {target.id: true}}} : {'type': 'skip'}", Event: "MetadataChanged"},
				},
			},
		},
	}
	reg, _ := rules.NewRegistry(manifest, func(s string) int { return 10 }, nil)

	params := map[string]any{"weapon": "longsword", "opportunity": true}
	events, _ := ExecuteGenericCommand("opportunity_attack", "thorne", []string{"elara"}, params, "", state, loader, reg)
	events[0].Apply(state)
	state.Metadata["pending_adjudication"] = map[string]any{"approved": true}
	events, _ = ExecuteGenericCommand("opportunity_attack", "thorne", []string{"elara"}, params, "", state, loader, reg)

	for _, e := range events {
		e.Apply(state)
	}
	assert.Equal(t, 1, state.Entities["thorne"].Spent["reactions"])
}

type mockStore struct{}

func (m *mockStore) Append(e engine.Event) error   { return nil }
func (m *mockStore) Load() ([]engine.Event, error) { return nil, nil }
func (m *mockStore) Close() error                  { return nil }
