package session

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suderio/ancient-draconic/internal/engine"
)

func testSession(t *testing.T) (*Session, string) {
	t.Helper()
	dir := t.TempDir()
	storePath := filepath.Join(dir, "test.jsonl")

	eval, err := engine.NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)

	store, err := NewStore(storePath)
	require.NoError(t, err)

	s := &Session{
		manifest: &engine.Manifest{
			Commands: map[string]engine.CommandDef{
				"encounter_start": {
					Name: "encounter start",
					Game: []engine.GameStep{
						{Name: "create_loop", Value: "loop('encounter_start', true)"},
						{Name: "order_loop", Value: "loop_order('encounter_start', false)"},
					},
				},
				"encounter_end": {
					Name: "encounter end",
					Game: []engine.GameStep{
						{Name: "state_change", Value: "loop('encounter_start', false)"},
					},
				},
			},
			Restrictions: engine.Restrictions{
				GMCommands: []string{"encounter_start", "encounter_end"},
			},
		},
		state: engine.NewGameState(),
		store: store,
		eval:  eval,
	}

	return s, storePath
}

func TestUndoSingleEvent(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)
	assert.True(t, s.State().IsLoopActive("encounter_start"))

	events, err := s.Execute("undo")
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Contains(t, events[0].Message(), "Undid 1 event")

	// After undoing 1 event, the loop order event is gone but the loop itself may still exist
	// Let's undo everything
	_, err = s.Execute("undo")
	require.NoError(t, err)
	assert.False(t, s.State().IsLoopActive("encounter_start"))
}

func TestUndoMultiple(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)
	assert.True(t, s.State().IsLoopActive("encounter_start"))

	count, _ := s.store.EventCount()
	events, err := s.Execute("undo steps: " + itoa(count))
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "event")

	assert.False(t, s.State().IsLoopActive("encounter_start"))
}

func TestUndoMoreThanExists(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	_, err = s.Execute("undo steps: 100")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot undo 100 events")
}

func TestUndoPersistence(t *testing.T) {
	s, storePath := testSession(t)

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	_, err = s.Execute("encounter end")
	require.NoError(t, err)

	// Undo the encounter_end (1 event)
	_, err = s.Execute("undo")
	require.NoError(t, err)
	assert.True(t, s.State().IsLoopActive("encounter_start"))
	s.Close()

	// Reopen and verify the undo persisted
	eval2, err := engine.NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)
	store2, err := NewStore(storePath)
	require.NoError(t, err)
	defer store2.Close()

	events, err := store2.Load()
	require.NoError(t, err)

	state := engine.NewGameState()
	for _, evt := range events {
		require.NoError(t, evt.Apply(state))
	}

	// The encounter_end event should be gone — loop should still be active
	assert.True(t, state.IsLoopActive("encounter_start"))
	_ = eval2
}

func TestUndoGMOnly(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	_, err = s.Execute("undo by: fighter")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestUndoWithCustomEvent(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()

	// Add a command that uses emit()
	s.manifest.Commands["test_spell"] = engine.CommandDef{
		Name: "test spell",
		Game: []engine.GameStep{
			{Name: "cast", Value: "emit('fire_bolt', {damage = 42})"},
		},
	}

	_, err := s.Execute("test_spell")
	require.NoError(t, err)
	assert.NotNil(t, s.State().Metadata["fire_bolt"])

	_, err = s.Execute("undo")
	require.NoError(t, err)
	assert.Nil(t, s.State().Metadata["fire_bolt"])
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func testSessionWithEndTurn(t *testing.T) (*Session, string) {
	t.Helper()
	s, storePath := testSession(t)

	// Add turn command
	s.manifest.Commands["turn"] = engine.CommandDef{
		Name: "turn",
		Game: []engine.GameStep{
			{Name: "advance", Value: "next_turn('encounter_start')"},
		},
	}

	return s, storePath
}

func TestNextTurn_AdvancesTurn(t *testing.T) {
	s, _ := testSessionWithEndTurn(t)
	defer s.Close()

	// Start encounter
	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	// Manually add actors and set order
	loop := s.State().Loops["encounter_start"]
	loop.Actors = []string{"fighter", "wizard", "rogue"}
	loop.Order = map[string]int{"fighter": 20, "wizard": 12, "rogue": 15}

	// End turn should emit TurnEnded, RoundStarted (first round), TurnStarted
	events, err := s.Execute("turn")
	require.NoError(t, err)

	// Should have: TurnEnded, RoundStarted (round 0→1), TurnStarted
	var types []string
	for _, e := range events {
		types = append(types, e.Type())
	}
	assert.Contains(t, types, "TurnEndedEvent")
	assert.Contains(t, types, "TurnStartedEvent")
	assert.Contains(t, types, "RoundStartedEvent")
}

func TestNextTurn_RoundWraparound(t *testing.T) {
	s, _ := testSessionWithEndTurn(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	loop := s.State().Loops["encounter_start"]
	loop.Actors = []string{"fighter", "wizard"}
	loop.Order = map[string]int{"fighter": 20, "wizard": 12}

	// First end turn: round 1 starts, fighter goes
	_, err = s.Execute("turn")
	require.NoError(t, err)
	assert.Equal(t, 1, s.State().Loops["encounter_start"].Round)

	// Second end turn: wizard goes
	_, err = s.Execute("turn")
	require.NoError(t, err)

	// Third end turn: wrap around → round 2
	_, err = s.Execute("turn")
	require.NoError(t, err)
	assert.Equal(t, 2, s.State().Loops["encounter_start"].Round)
}

func TestUndoToBoundary_Turn(t *testing.T) {
	s, _ := testSessionWithEndTurn(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	loop := s.State().Loops["encounter_start"]
	loop.Actors = []string{"fighter", "wizard"}
	loop.Order = map[string]int{"fighter": 20, "wizard": 12}

	// Do 2 end turns
	_, err = s.Execute("turn")
	require.NoError(t, err)
	_, err = s.Execute("turn")
	require.NoError(t, err)

	// Undo to previous turn boundary
	events, err := s.Execute("undo turn: 1")
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "TurnStartedEvent")
}

func TestUndoToBoundary_Round(t *testing.T) {
	s, _ := testSessionWithEndTurn(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	loop := s.State().Loops["encounter_start"]
	loop.Actors = []string{"fighter"}
	loop.Order = map[string]int{"fighter": 20}

	// Do 3 end turns to create 3 rounds (1 actor = 1 turn per round)
	for i := 0; i < 3; i++ {
		_, err = s.Execute("turn")
		require.NoError(t, err)
	}
	assert.Equal(t, 3, s.State().Loops["encounter_start"].Round)

	// Undo to previous round
	events, err := s.Execute("undo round: 1")
	require.NoError(t, err)
	assert.Contains(t, events[0].Message(), "RoundStartedEvent")
}

func TestUndoToBoundary_NotEnough(t *testing.T) {
	s, _ := testSessionWithEndTurn(t)
	defer s.Close()

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	_, err = s.Execute("undo turn: 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only 0 found")
}

func TestParseIntParam(t *testing.T) {
	assert.Equal(t, 42, parseIntParam(42, 1))
	assert.Equal(t, 42, parseIntParam(float64(42), 1))
	assert.Equal(t, 42, parseIntParam("42", 1))
	assert.Equal(t, 1, parseIntParam("invalid", 1))
	assert.Equal(t, 1, parseIntParam(nil, 1))
	assert.Equal(t, 1, parseIntParam(true, 1))
}
