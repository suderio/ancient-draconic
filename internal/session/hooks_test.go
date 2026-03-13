package session

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suderio/ancient-draconic/internal/engine"
)

func testHooksSession(t *testing.T) (*Session, string) {
	t.Helper()
	dir := t.TempDir()
	storePath := filepath.Join(dir, "hooks_test.jsonl")

	eval, err := engine.NewLuaEvaluator(func(dice string) int { return 10 })
	require.NoError(t, err)

	store, err := NewStore(storePath)
	require.NoError(t, err)

	s := &Session{
		manifest: &engine.Manifest{
			Commands: map[string]engine.CommandDef{
				"encounter_start": {
					Name: "encounter start",
					Game: engine.CommandPhase{Steps: []engine.GameStep{
						{Name: "create_loop", Value: "loop('encounter_start', true)"},
					}},
				},
				"turn": {
					Name: "turn",
					Game: engine.CommandPhase{Steps: []engine.GameStep{
						{Name: "advance", Value: "next_turn('encounter_start')"},
					}},
				},
				"disengage": {
					Name: "disengage",
					Actor: engine.CommandPhase{
						Steps: []engine.GameStep{
							{Name: "disengage_apply", Value: "condition('disengaged')"},
						},
						Hooks: []engine.HookDef{
							{Name: "end_disengage", Type: "next_turn", Value: "remove_condition('disengaged')"},
						},
					},
				},
			},
		},
		state: engine.NewGameState(),
		store: store,
		eval:  eval,
	}

	// Setup initial actors
	s.state.Entities["fighter"] = engine.NewEntity("fighter", "Fighter")
	s.state.Entities["wizard"] = engine.NewEntity("wizard", "Wizard")

	return s, storePath
}

func TestHooks_NextTurn(t *testing.T) {
	s, storePath := testHooksSession(t)
	defer s.Close()
	_ = storePath

	_, err := s.Execute("encounter start")
	require.NoError(t, err)

	loop := s.State().Loops["encounter_start"]
	loop.Actors = []string{"fighter", "wizard"}
	loop.Order = map[string]int{"fighter": 20, "wizard": 10}

	// Start the first turn directly to test
	_, err = s.Execute("turn")
	require.NoError(t, err)
	// Currently wizard's turn (since index 0 advanced to 1)
	assert.Equal(t, "wizard", s.State().Loops["encounter_start"].Actors[s.State().Loops["encounter_start"].Current])

	// Wizard uses disengage
	_, err = s.Execute("disengage by: wizard")
	require.NoError(t, err)

	// Wizard should have 'disengaged' condition
	assert.Contains(t, s.State().Entities["wizard"].Conditions, "disengaged")

	// Check hooks are registered
	assert.Len(t, s.State().Entities["wizard"].Hooks, 1)

	// Advance turn to fighter
	_, err = s.Execute("turn")
	require.NoError(t, err)
	assert.Equal(t, "fighter", s.State().Loops["encounter_start"].Actors[s.State().Loops["encounter_start"].Current])

	// Next turn has happened in the loop. The hook 'next_turn' should have evaluated.
	// Since 'next_turn' is universal, ANY turn starting will trigger it.
	// The wizard should NO LONGER have 'disengaged'.
	assert.NotContains(t, s.State().Entities["wizard"].Conditions, "disengaged")

	// The hook should be removed
	assert.Len(t, s.State().Entities["wizard"].Hooks, 0)
}
