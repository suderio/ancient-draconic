package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockRoll(dice string) int { return 10 }

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
					{Name: "check_conflict", Value: "not is_encounter_start_active()", Error: "an encounter is already active"},
				},
				Hint:  "Roll initiative for all actors.",
				Help:  "Starts an encounter.",
				Error: "encounter start [with: Target1]",
				Game: CommandPhase{Steps: []GameStep{
					{Name: "create_loop", Value: "loop('encounter_start', true)"},
					{Name: "order_loop", Value: "loop_order('encounter_start', false)"},
				}},
				Targets: CommandPhase{Steps: []GameStep{
					{Name: "ask_initiative", Value: "ask(target.id, 'initiative')"},
				}},
			},
			"encounter_end": {
				Name: "encounter end",
				Prereq: []PrereqStep{
					{Name: "check_conflict", Value: "is_encounter_start_active()", Error: "no active encounter to end"},
				},
				Hint:  "Encounter has ended.",
				Help:  "Ends an encounter.",
				Error: "encounter end",
				Game: CommandPhase{Steps: []GameStep{
					{Name: "state_change", Value: "loop('encounter_start', false)"},
				}},
			},
			"initiative": {
				Name: "initiative",
				Prereq: []PrereqStep{
					{Name: "check_active", Value: "is_encounter_start_active()", Error: "an encounter is not active"},
				},
				Hint:  "Wait for your turn.",
				Help:  "Rolls initiative.",
				Error: "initiative",
				Game: CommandPhase{Steps: []GameStep{
					{Name: "roll_score", Value: "loop_value('encounter_start', 10 + ((actor.stats.dex / 2) - 5))"},
				}},
			},
			"grapple": {
				Name: "grapple",
				Params: []ParamDef{
					{Name: "to", Type: "target", Required: true},
				},
				Prereq: []PrereqStep{
					{Name: "check_action", Value: "actor.spent.actions < actor.resources.actions", Error: "no actions remaining"},
				},
				Hint:  "Grapple command grapples the target.",
				Help:  "Grapple command grapples the target.",
				Error: "grapple [to: <target>]",
				Game: CommandPhase{Steps: []GameStep{
					{Name: "contest", Value: "contest(10 + ((actor.stats.str / 2) - 5))"},
				}},
				Targets: CommandPhase{Steps: []GameStep{
					{Name: "grappled", Value: "condition('grappled')"},
				}},
				Actor: CommandPhase{Steps: []GameStep{
					{Name: "consume_action", Value: "spend('actions')"},
				}},
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
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	_, err = ExecuteCommand("encounter_end", "GM", nil, nil, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active encounter to end")
}

func TestParamValidation(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	_, err = ExecuteCommand("grapple", "fighter", nil, map[string]any{}, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter: to")
}

func TestGMRestriction(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	_, err = ExecuteCommand("encounter_start", "fighter", nil, nil, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")

	events, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	assert.NoError(t, err)
	assert.NotEmpty(t, events)
}

func TestLoopLifecycle(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}
	assert.True(t, state.IsLoopActive("encounter_start"))

	events, err = ExecuteCommand("encounter_end", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}
	assert.False(t, state.IsLoopActive("encounter_start"))
}

func TestGameSteps(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("encounter_start", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		e.Apply(state)
	}

	// Roll initiative: 10 + (14/2 - 5) = 10 + 2 = 12
	events, err = ExecuteCommand("initiative", "fighter", nil, nil, state, m, eval)
	require.NoError(t, err)
	assert.NotEmpty(t, events)

	found := false
	for _, e := range events {
		if lo, ok := e.(*LoopOrderEvent); ok {
			assert.Equal(t, "fighter", lo.ActorID)
			assert.Equal(t, 12, lo.Value)
			found = true
		}
	}
	assert.True(t, found, "expected LoopOrderEvent")
}

func TestTargetIteration(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("encounter_start", "GM", nil,
		map[string]any{"with": []string{"fighter", "goblin"}}, state, m, eval)
	require.NoError(t, err)

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
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("grapple", "fighter", []string{"goblin"},
		map[string]any{"to": "goblin"}, state, m, eval)
	require.NoError(t, err)

	for _, e := range events {
		e.Apply(state)
	}

	assert.Equal(t, 1, state.Entities["fighter"].Spent["actions"])
	assert.Contains(t, state.Entities["goblin"].Conditions, "grappled")
}

func TestAskIssuedEvent(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

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
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("help", "GM", nil,
		map[string]any{"command": "encounter_start"}, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "Starts an encounter")

	events, err = ExecuteCommand("help", "GM", nil, map[string]any{}, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "Available commands")
}

func TestHintCommand(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("hint", "GM", nil, nil, state, m, eval)
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "No command has been executed")

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
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	state.Entities["fighter"].Spent["actions"] = 1

	_, err = ExecuteCommand("grapple", "fighter", []string{"goblin"},
		map[string]any{"to": "goblin"}, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions remaining")
}

func TestRollCommand(t *testing.T) {
	m := testManifest()
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
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

func TestCustomEvent(t *testing.T) {
	m := &Manifest{
		Commands: map[string]CommandDef{
			"my_spell": {
				Name: "my spell",
				Game: CommandPhase{Steps: []GameStep{
					{Name: "cast", Value: "emit('arcane_blast', {power = 42})"},
				}},
			},
		},
	}
	state := testState()
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	events, err := ExecuteCommand("my_spell", "fighter", nil, nil, state, m, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)

	ce, ok := events[0].(*CustomEvent)
	require.True(t, ok)
	assert.Equal(t, "arcane_blast", ce.EventType)
	assert.Equal(t, 42, ce.Payload["power"])

	ce.Apply(state)
	assert.Equal(t, 42, state.Metadata["arcane_blast"].(map[string]any)["power"])
}

func TestHelperFunctions(t *testing.T) {
	eval, err := NewLuaEvaluator(mockRoll)
	require.NoError(t, err)

	tests := []struct {
		expr     string
		expected string
	}{
		{"loop('enc', true)", "loop"},
		{"loop_order('enc', false)", "loop_order"},
		{"loop_value('enc', 15)", "loop_value"},
		{"add_actor('fighter')", "add_actor"},
		{"ask('fighter', 'initiative')", "ask"},
		{"condition('grappled')", "condition"},
		{"remove_condition('grappled')", "condition"},
		{"spend('actions')", "spend"},
		{"set_attr('stats', 'hp', 10)", "set_attr"},
		{"contest(15)", "contest"},
		{"check_result(true)", "check"},
		{"hint('hello')", "hint"},
		{"metadata('key', 'val')", "metadata"},
		{"emit('my_event', {x = 1})", "my_event"},
	}

	for _, tt := range tests {
		result, err := eval.Eval(tt.expr, nil)
		require.NoError(t, err, "expr: %s", tt.expr)
		m, ok := result.(map[string]any)
		require.True(t, ok, "expr: %s returned %T", tt.expr, result)
		assert.Equal(t, tt.expected, m["_event"], "expr: %s", tt.expr)
	}
}

func TestExecuteCommand_MoveAndDash(t *testing.T) {
	eval, err := NewLuaEvaluator(nil)
	require.NoError(t, err)
	defer eval.Close()

	m, err := eval.LoadManifestLua("../../test/manifest.lua")
	require.NoError(t, err)

	state := NewGameState()
	actor := NewEntity("fighter", "Fighter")
	actor.Resources["actions"] = 3 // Give 3 actions for testing multiple dashes
	actor.Resources["speed"] = 30
	state.Entities["fighter"] = actor

	// 1. Move 30 feet (all movement)
	events, err := ExecuteCommand("move", "fighter", nil, map[string]any{"feet": 30}, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}
	assert.Equal(t, 30, actor.Spent["speed"])

	// 2. Dash (resets spent movement)
	events, err = ExecuteCommand("dash", "fighter", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}
	assert.Equal(t, 1, actor.Spent["actions"])
	assert.Equal(t, 0, actor.Spent["speed"]) // Reset to 0

	// 3. Move 30 more feet
	events, err = ExecuteCommand("move", "fighter", nil, map[string]any{"feet": 30}, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}
	assert.Equal(t, 30, actor.Spent["speed"])

	// 4. Dash again
	events, err = ExecuteCommand("dash", "fighter", nil, nil, state, m, eval)
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, e.Apply(state))
	}
	assert.Equal(t, 2, actor.Spent["actions"])
	assert.Equal(t, 0, actor.Spent["speed"])
}

func TestExecuteCommand_SpendCustomAmount(t *testing.T) {
	eval, err := NewLuaEvaluator(nil)
	require.NoError(t, err)
	defer eval.Close()

	state := NewGameState()
	actor := NewEntity("fighter", "Fighter")
	state.Entities["fighter"] = actor

	// Direct test of spend helper via Eval
	result, err := eval.Eval("spend('arrows', 5)", nil)
	require.NoError(t, err)

	// dispatchTaggedResult to convert to Event
	evts, _ := dispatchTaggedResult(result, "fighter", "", "test", state)
	require.Len(t, evts, 1)

	evt := evts[0].(*AddSpentEvent)
	assert.Equal(t, "arrows", evt.Key)
	assert.Equal(t, 5, evt.Amount)

	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 5, actor.Spent["arrows"])
}
