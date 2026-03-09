package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteBuiltin_UnknownCommand(t *testing.T) {
	state := NewGameState()
	m := &Manifest{Commands: map[string]CommandDef{}}
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	_, err = executeBuiltin("nonexistent", "GM", nil, nil, state, m, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown builtin command")
}

func TestExecuteRoll_Success(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 15 })
	require.NoError(t, err)
	defer eval.Close()

	events, err := executeRoll("fighter", map[string]any{"dice": "1d20"}, eval)
	require.NoError(t, err)
	require.Len(t, events, 1)
	dre, ok := events[0].(*DiceRolledEvent)
	require.True(t, ok)
	assert.Equal(t, "fighter", dre.ActorID)
	assert.Equal(t, "1d20", dre.Dice)
	assert.Equal(t, 15, dre.Result)
}

func TestExecuteRoll_MissingDice(t *testing.T) {
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	_, err = executeRoll("fighter", map[string]any{}, eval)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dice")

	_, err = executeRoll("fighter", map[string]any{"dice": ""}, eval)
	assert.Error(t, err)
}

func TestExecuteHelp_AllCommands(t *testing.T) {
	m := &Manifest{
		Commands: map[string]CommandDef{
			"attack": {Name: "attack", Help: "Perform an attack", Error: "attack target: <name>"},
		},
	}

	events, err := executeHelp(map[string]any{}, m)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "Available commands")
	assert.Contains(t, events[0].Message(), "attack")
}

func TestExecuteHelp_SpecificCommand(t *testing.T) {
	m := &Manifest{
		Commands: map[string]CommandDef{
			"attack": {Name: "attack", Help: "Perform an attack", Error: "attack target: <name>"},
		},
	}

	events, err := executeHelp(map[string]any{"command": "attack"}, m)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "attack")
}

func TestExecuteHelp_SpecificCommandUnderscore(t *testing.T) {
	m := &Manifest{
		Commands: map[string]CommandDef{
			"encounter_start": {Name: "encounter start", Help: "Start encounter", Error: "encounter start"},
		},
	}

	events, err := executeHelp(map[string]any{"command": "encounter start"}, m)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "encounter start")
}

func TestExecuteHelp_UnknownCommand(t *testing.T) {
	m := &Manifest{Commands: map[string]CommandDef{}}
	_, err := executeHelp(map[string]any{"command": "fly"}, m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestExecuteHint_NoCommand(t *testing.T) {
	state := NewGameState()
	m := &Manifest{Commands: map[string]CommandDef{}}

	events, err := executeHint(state, m)
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "No command")
}

func TestExecuteHint_WithHint(t *testing.T) {
	state := NewGameState()
	state.LastCommand = "attack"
	m := &Manifest{
		Commands: map[string]CommandDef{
			"attack": {Name: "attack", Hint: "Remember to add modifiers"},
		},
	}

	events, err := executeHint(state, m)
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "Remember to add modifiers")
}

func TestExecuteHint_NoHintForCommand(t *testing.T) {
	state := NewGameState()
	state.LastCommand = "attack"
	m := &Manifest{
		Commands: map[string]CommandDef{
			"attack": {Name: "attack", Hint: ""},
		},
	}

	events, err := executeHint(state, m)
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "No hint")
}

func TestExecuteAsk_Success(t *testing.T) {
	events, err := executeAsk("GM", []string{"player1", "player2"}, map[string]any{
		"options": []any{"attack", "defend"},
	})
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestExecuteAsk_NoTargets(t *testing.T) {
	_, err := executeAsk("GM", nil, map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least one target")
}

func TestExecuteAsk_NoOptions(t *testing.T) {
	events, err := executeAsk("GM", []string{"player1"}, map[string]any{})
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestExecuteAllow_Success(t *testing.T) {
	state := NewGameState()
	state.Metadata["pending_ask"] = map[string]any{"target": "player1"}

	events, err := executeAllow("GM", state)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Nil(t, state.Metadata["pending_ask"])
}

func TestExecuteAllow_NotGM(t *testing.T) {
	state := NewGameState()
	_, err := executeAllow("player1", state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only the GM")
}

func TestExecuteDeny_Success(t *testing.T) {
	state := NewGameState()
	state.Metadata["pending_ask"] = map[string]any{"target": "player1"}

	events, err := executeDeny("GM", state)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Nil(t, state.Metadata["pending_ask"])
}

func TestExecuteDeny_NotGM(t *testing.T) {
	state := NewGameState()
	_, err := executeDeny("player1", state)
	assert.Error(t, err)
}

func TestExecuteAdjudicate(t *testing.T) {
	state := NewGameState()
	events, err := executeAdjudicate("GM", state)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestIsBuiltin(t *testing.T) {
	assert.True(t, isBuiltin("roll"))
	assert.True(t, isBuiltin("help"))
	assert.True(t, isBuiltin("hint"))
	assert.True(t, isBuiltin("ask"))
	assert.True(t, isBuiltin("allow"))
	assert.True(t, isBuiltin("deny"))
	assert.True(t, isBuiltin("adjudicate"))
	assert.True(t, isBuiltin("undo"))
	assert.False(t, isBuiltin("attack"))
}

func TestExecuteBuiltin_AllCommands(t *testing.T) {
	state := NewGameState()
	state.Metadata["pending_ask"] = true
	m := &Manifest{Commands: map[string]CommandDef{
		"attack": {Name: "attack", Help: "X", Hint: "Y"},
	}}
	eval, err := NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	defer eval.Close()

	tests := []struct {
		cmd     string
		actor   string
		targets []string
		params  map[string]any
	}{
		{"roll", "fighter", nil, map[string]any{"dice": "1d20"}},
		{"help", "GM", nil, map[string]any{}},
		{"hint", "GM", nil, nil},
		{"ask", "GM", []string{"player1"}, map[string]any{}},
		{"allow", "GM", nil, nil},
		{"deny", "GM", nil, nil},
		{"adjudicate", "GM", nil, nil},
		{"undo", "GM", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			events, err := executeBuiltin(tt.cmd, tt.actor, tt.targets, tt.params, state, m, eval)
			require.NoError(t, err)
			assert.NotEmpty(t, events)
		})
	}
}
