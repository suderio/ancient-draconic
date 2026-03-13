package session

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suderio/ancient-draconic/internal/engine"
)

func TestHideAndConditions(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "hide_test.jsonl")

	// Create evaluator with fixed dice roll (we want 15 total, so roll 12 + 3 dex = 15)
	eval, err := engine.NewLuaEvaluator(func(dice string) int { return 12 })
	require.NoError(t, err)

	store, err := NewStore(storePath)
	require.NoError(t, err)

	// Since we migrated test/manifest.lua and world/dnd5e/manifest.lua, we can just load test/manifest.lua
	manifest, err := eval.LoadManifestLua("../../test/manifest.lua")
	require.NoError(t, err)

	s := &Session{
		manifest: manifest,
		state:    engine.NewGameState(),
		store:    store,
		eval:     eval,
	}

	// Setup rogue actor with dexterity and stealth proficiency
	rogue := engine.NewEntity("rogue", "Rogue")
	rogue.Stats["dex"] = 16 // modifier +3
	rogue.Stats["prof_bonus"] = 2
	rogue.Proficiencies["stealth"] = 1 // 1 * prof_bonus = +2 // total bonus = +5. Roll of 12 + 5 = 17 >= 15 (Success)
	rogue.Resources["actions"] = 1
	s.state.Entities["rogue"] = rogue

	// Setup GM actor to test remove_condition
	gm := engine.NewEntity("GM", "GM")
	s.state.Entities["GM"] = gm

	// Execute Hide command
	hints, err := s.Execute("hide by: rogue")
	require.NoError(t, err)

	// Check if the condition "invisible" is applied
	assert.Contains(t, s.State().Entities["rogue"].Conditions, "invisible")
	
	// Check the returned hints (should include "Success - Use as DC to find")
	foundHint := false
	for _, h := range hints {
		if strings.Contains(h.Message(), "Stealth check total:") && strings.Contains(h.Message(), "Success") {
			foundHint = true
		}
	}
	assert.True(t, foundHint, "Expected a success hint for the stealth check")

	// Execute GM add_condition Command
	_, err = s.Execute("add condition condition: blinded to: rogue by: GM")
	require.NoError(t, err)

	assert.Contains(t, s.State().Entities["rogue"].Conditions, "blinded")

	// Execute GM remove_condition Command to remove invisible
	_, err = s.Execute("remove condition condition: invisible from: rogue by: GM")
	require.NoError(t, err)

	assert.NotContains(t, s.State().Entities["rogue"].Conditions, "invisible")
	assert.Contains(t, s.State().Entities["rogue"].Conditions, "blinded")
}

func TestHideFailure(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "hide_fail_test.jsonl")

	// Roll 5. 5 + 3 (dex) = 8 < 15.
	eval, err := engine.NewLuaEvaluator(func(dice string) int { return 5 })
	require.NoError(t, err)

	store, err := NewStore(storePath)
	require.NoError(t, err)

	manifest, err := eval.LoadManifestLua("../../test/manifest.lua")
	require.NoError(t, err)

	s := &Session{
		manifest: manifest,
		state:    engine.NewGameState(),
		store:    store,
		eval:     eval,
	}

	rogue := engine.NewEntity("rogue", "Rogue")
	rogue.Stats["dex"] = 16 // +3
	rogue.Resources["actions"] = 1
	s.state.Entities["rogue"] = rogue

	hints, err := s.Execute("hide by: rogue")
	require.NoError(t, err)

	// Check if the condition "invisible" is NOT applied
	assert.NotContains(t, s.State().Entities["rogue"].Conditions, "invisible")
	
	foundHint := false
	for _, h := range hints {
		if strings.Contains(h.Message(), "Stealth check total: 8") && strings.Contains(h.Message(), "Failure") {
			foundHint = true
		}
	}
	assert.True(t, foundHint, "Expected a failure hint for the stealth check")
}
