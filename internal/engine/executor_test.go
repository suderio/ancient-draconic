package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoll returns a deterministic value for testing.
func mockRoll(dice string) int { return 10 }

// testManifest creates a small manifest for testing with encounter and grapple commands.
func testManifest() *Manifest {
	return &Manifest{
		Restrictions: Restrictions{
			Adjudication: struct {
				Commands []string `yaml:"commands"`
			}{Commands: []string{"grapple"}},
			GMCommands: []string{"encounter_start", "encounter_end"},
		},
		Commands: map[string]CommandDef{
			"encounter_start": {
				Name: "encounter start",
				Params: []ParamDef{
					{Name: "with", Type: "list<target>", Required: false},
				},
				Prereq: []PrereqStep{
					{Name: "check_conflict", Formula: "!is_encounter_start_active", Error: "an encounter is already active"},
				},
				Hint:  "Roll initiative for all actors.",
				Help:  "Starts an encounter.",
				Error: "encounter start [with: Target1]",
				Game: []GameStep{
					{Name: "create_loop", Formula: "true", Event: "LoopEvent"},
					{Name: "order_loop", Formula: "false", Event: "LoopOrderAscendingEvent"},
				},
				Targets: []GameStep{
					{Name: "ask_initiative", Formula: "[target.id, 'initiative']", Event: "AskIssuedEvent"},
				},
			},
			"encounter_end": {
				Name: "encounter end",
				Prereq: []PrereqStep{
					{Name: "check_conflict", Formula: "is_encounter_start_active", Error: "no active encounter to end"},
				},
				Hint:  "Encounter has ended.",
				Help:  "Ends an encounter.",
				Error: "encounter end",
				Game: []GameStep{
					{Name: "state_change", Formula: "false", Event: "LoopEvent", Loop: "encounter_start"},
				},
			},
			"initiative": {
				Name: "initiative",
				Prereq: []PrereqStep{
					{Name: "check_active", Formula: "is_encounter_start_active", Error: "an encounter is not active"},
				},
				Hint:  "Wait for your turn.",
				Help:  "Rolls initiative.",
				Error: "initiative",
				Game: []GameStep{
					{Name: "roll_score", Formula: "roll('1d20') + mod(actor.stats.dex)", Event: "LoopOrderEvent"},
				},
			},
			"grapple": {
				Name: "grapple",
				Params: []ParamDef{
					{Name: "to", Type: "target", Required: true},
				},
				Prereq: []PrereqStep{
					{Name: "check_action", Formula: "actor.spent.actions < actor.resources.actions", Error: "no actions remaining"},
				},
				Hint:  "Grapple command grapples the target.",
				Help:  "Grapple command grapples the target.",
				Error: "grapple [to: <target>]",
				Game: []GameStep{
					{Name: "contest", Formula: "roll('1d20') + mod(actor.stats.str)", Event: "ContestStarted"},
				},
				Targets: []GameStep{
					{Name: "grappled", Formula: "'grappled'", Event: "AddConditionEvent"},
				},
				Actor: []GameStep{
					{Name: "consume_action", Formula: "'actions'", Event: "AddSpentEvent"},
				},
			},
		},
	}
}

func testState() *GameState {
	state := NewGameState()
	state.Entities["fighter"] = NewEntity("fighter", "Fighter")
	state.Entities["fighter"].Stats["str"] = 18
	state.Entities["fighter"].Stats["dex"] = 14
	state.Entities["fighter"].Resources["actions"] = 1
	state.Entities["fighter"].Spent["actions"] = 0

	state.Entities["goblin"] = NewEntity("goblin", "Goblin")
	state.Entities["goblin"].Stats["str"] = 8
	state.Entities["goblin"].Stats["dex"] = 14
	state.Entities["goblin"].Stats["ac"] = 15
	return state
}

func TestPrereqValidation(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// encounter_end should fail: no active encounter
	_, err = ExecuteCommand("encounter_end", "GM", nil, nil, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active encounter to end")
}

func TestParamValidation(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// grapple requires "to" param
	_, err = ExecuteCommand("grapple", "fighter", nil, map[string]any{}, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter: to")
}

func TestGMRestriction(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// encounter_start by non-GM should fail
	_, err = ExecuteCommand("encounter_start", "fighter", nil, nil, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")

	// encounter_start by GM should succeed
	events, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	assert.NoError(t, err)
	assert.NotEmpty(t, events)
}

func TestLoopLifecycle(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Start encounter
	events, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)

	// Apply all events
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}

	// Loop should be active
	assert.True(t, state.IsLoopActive("encounter_start"))

	// End encounter
	events, err = ExecuteCommand("encounter_end", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}

	// Loop should be inactive
	assert.False(t, state.IsLoopActive("encounter_start"))
}

func TestGameSteps(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Start encounter first
	events, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	// Roll initiative (game step with formula referencing actor stats)
	// mockRoll returns 10, mod(14) = 2, so result = 12
	events, err = ExecuteCommand("initiative", "fighter", nil, nil, state, m, eval)
	require.NoError(t, err)
	assert.NotEmpty(t, events)

	// Should have a LoopOrderEvent
	found := false
	for _, e := range events {
		if lo, ok := e.(*LoopOrderEvent); ok {
			assert.Equal(t, "fighter", lo.ActorID)
			assert.Equal(t, 12, lo.Value) // 10 + mod(14) = 10 + 2 = 12
			found = true
		}
	}
	assert.True(t, found, "expected LoopOrderEvent")
}

func TestTargetIteration(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Start encounter with targets
	events, err := ExecuteCommand("encounter_start", "GM", nil,
		map[string]any{"with": []string{"fighter", "goblin"}}, state, m, eval)
	require.NoError(t, err)

	// Should have AskIssuedEvent for each target
	askCount := 0
	for _, e := range events {
		if ask, ok := e.(*AskIssuedEvent); ok {
			askCount++
			assert.Contains(t, []string{"fighter", "goblin"}, ask.TargetID)
			assert.Equal(t, []string{"initiative"}, ask.Options)
		}
	}
	assert.Equal(t, 2, askCount, "expected one AskIssuedEvent per target")
}

func TestActorSteps(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Grapple: actor steps should consume an action
	events, err := ExecuteCommand("grapple", "fighter", []string{"goblin"},
		map[string]any{"to": "goblin"}, state, m, eval)
	require.NoError(t, err)

	// Apply events
	for _, e := range events {
		e.Apply(state)
	}

	// Actor should have spent an action
	assert.Equal(t, 1, state.Entities["fighter"].Spent["actions"])

	// Target should have "grappled" condition
	assert.Contains(t, state.Entities["goblin"].Conditions, "grappled")
}

func TestAskIssuedEvent(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Use "ask" hardcoded command
	events, err := ExecuteCommand("ask", "GM", []string{"fighter"},
		map[string]any{"options": []string{"check skill: athletics dc: 15"}}, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)

	ask, ok := events[0].(*AskIssuedEvent)
	require.True(t, ok)
	assert.Equal(t, "fighter", ask.TargetID)
	assert.Equal(t, []string{"check skill: athletics dc: 15"}, ask.Options)
}

func TestHelpCommand(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Help for specific command
	events, err := ExecuteCommand("help", "GM", nil,
		map[string]any{"command": "encounter_start"}, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "Starts an encounter")

	// General help
	events, err = ExecuteCommand("help", "GM", nil, map[string]any{}, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "Available commands")
}

func TestHintCommand(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// No command executed yet
	events, err := ExecuteCommand("hint", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "No command has been executed")

	// After starting encounter, hint should show encounter hint
	startEvents, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range startEvents {
		e.Apply(state)
	}

	events, err = ExecuteCommand("hint", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "Roll initiative")
}

func TestPrereqBlocksExecution(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	// Exhaust actions
	state.Entities["fighter"].Spent["actions"] = 1

	// Grapple should fail: no actions remaining
	_, err = ExecuteCommand("grapple", "fighter", []string{"goblin"},
		map[string]any{"to": "goblin"}, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions remaining")
}

func TestRollCommand(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("roll", "fighter", nil,
		map[string]any{"dice": "1d20"}, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)

	roll, ok := events[0].(*DiceRolledEvent)
	require.True(t, ok)
	assert.Equal(t, "fighter", roll.ActorID)
	assert.Equal(t, "1d20", roll.Dice)
	assert.Equal(t, 10, roll.Result)
}
