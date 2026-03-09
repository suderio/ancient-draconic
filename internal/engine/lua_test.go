package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLuaEvaluator_Close(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	eval.Close()
}

func TestLuaEvaluator_DefaultRoll(t *testing.T) {
	eval, err := NewLuaEvaluator(nil)
	require.NoError(t, err)
	defer eval.Close()

	// defaultRoll uses rand.Intn — result is non-deterministic but within range
	result, err := eval.Eval("roll('1d6')", nil)
	require.NoError(t, err)
	// luaValueToGo returns int for integer Lua numbers
	v, ok := result.(int)
	require.True(t, ok, "expected int, got %T", result)
	assert.GreaterOrEqual(t, v, 1)
	assert.LessOrEqual(t, v, 6)
}

func TestLuaEvaluator_EvalString(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	// Eval auto-wraps strings with "return" so don't include it
	result, err := eval.Eval("2 + 3", nil)
	require.NoError(t, err)
	// luaValueToGo returns int for integer values
	assert.Equal(t, 5, result)
}

func TestLuaEvaluator_EvalWithContext(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	state := NewGameState()
	actor := NewEntity("fighter", "Fighter")
	actor.Stats["str"] = 18
	ctx := BuildContext(state, actor, nil, nil, nil, nil, nil)

	result, err := eval.Eval("actor.stats.str", ctx)
	require.NoError(t, err)
	assert.Equal(t, 18, result)
}

func TestLuaEvaluator_EvalNilResult(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	result, err := eval.Eval("nil", nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLuaEvaluator_EvalError(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	_, err = eval.Eval("error('boom')", nil)
	assert.Error(t, err)
}

func TestLuaEvaluator_EvalLiteralTypes(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	// int is a literal passthrough in Eval
	result, err := eval.Eval(42, nil)
	require.NoError(t, err)
	assert.Equal(t, 42, result)

	// float64 is a literal passthrough
	result, err = eval.Eval(3.14, nil)
	require.NoError(t, err)
	assert.Equal(t, 3.14, result)

	// bool is a literal passthrough
	result, err = eval.Eval(true, nil)
	require.NoError(t, err)
	assert.Equal(t, true, result)

	// Unsupported types should error
	_, err = eval.Eval([]string{"invalid"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestLoadManifestLua_Success(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	m, err := eval.LoadManifestLua("../../test/manifest.lua")
	require.NoError(t, err)
	assert.NotNil(t, m)
	assert.NotEmpty(t, m.Commands)
	assert.Contains(t, m.Commands, "encounter_start")
	assert.Contains(t, m.Commands, "encounter_start")
	assert.Contains(t, m.Commands, "turn")
}

func TestLoadManifestLua_FileNotFound(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	_, err = eval.LoadManifestLua("/nonexistent/manifest.lua")
	assert.Error(t, err)
}

func TestEntityToMap(t *testing.T) {
	e := NewEntity("fighter", "Fighter")
	e.Stats["str"] = 18
	e.Resources["hp"] = 45
	e.Conditions = append(e.Conditions, "poisoned")

	m := entityToMap(e)
	assert.Equal(t, "fighter", m["id"])
	assert.Equal(t, "Fighter", m["name"])
	// entityToMap preserves map[string]int as-is — check via actual type
	statsMap := m["stats"].(map[string]int)
	assert.Equal(t, 18, statsMap["str"])
}

func TestEntityToMap_Nil(t *testing.T) {
	m := entityToMap(nil)
	assert.Nil(t, m)
}

func TestBuildContext_Comprehensive(t *testing.T) {
	state := NewGameState()
	state.Loops["combat"] = &Loop{Active: true, Order: make(map[string]int)}

	actor := NewEntity("fighter", "Fighter")
	state.Entities["fighter"] = actor

	target := NewEntity("goblin", "Goblin")
	state.Entities["goblin"] = target

	params := map[string]any{"dc": 15}
	gameResults := map[string]any{"roll": 18}
	targetResults := map[string]any{"save": true}
	actorResults := map[string]any{"spent": 1}

	ctx := BuildContext(state, actor, target, params, gameResults, targetResults, actorResults)
	assert.NotNil(t, ctx)
	actorMap := ctx["actor"].(map[string]any)
	assert.Equal(t, "fighter", actorMap["id"])
	targetMap := ctx["target"].(map[string]any)
	assert.Equal(t, "goblin", targetMap["id"])
	f, ok := ctx["is_combat_active"].(func() any)
	assert.True(t, ok)
	assert.True(t, f().(bool))
}

func TestLuaHelpers_AllHelpers(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	tests := []struct {
		name   string
		expr   string
		evtKey string
	}{
		{"loop", "loop('combat', true)", "loop"},
		{"loop_order", "loop_order('combat', false)", "loop_order"},
		{"loop_value", "loop_value('combat', 18)", "loop_value"},
		{"add_actor", "add_actor('combat', 'fighter')", "add_actor"},
		{"ask", "ask('player1', {'attack', 'defend'})", "ask"},
		{"condition", "condition('poisoned', true)", "condition"},
		{"remove_condition", "remove_condition('poisoned')", "condition"},
		{"spend", "spend('spell_slot')", "spend"},
		{"set_attr", "set_attr('stats', 'str', 20)", "set_attr"},
		{"contest", "contest(18)", "contest"},
		{"check_result", "check_result(true)", "check"},
		{"hint", "hint('hello')", "hint"},
		{"metadata", "metadata('round', 3)", "metadata"},
		{"emit", "emit('fire_bolt', {damage = 42})", "fire_bolt"},
		{"next_turn", "next_turn('combat')", "next_turn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Eval(tt.expr, nil)
			require.NoError(t, err, "expr: %s", tt.expr)
			m, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map, got: %T", result)
			assert.Equal(t, tt.evtKey, m["_event"], "expected _event=%s", tt.evtKey)
		})
	}
}
