package command

import (
	"path/filepath"
	"testing"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
)

func TestExecuteEncounterStateValidations(t *testing.T) {
	state := engine.NewGameState()
	baseDir := filepath.Join("..", "..") // Assume root is two levels up from command/ for test files
	loader := data.NewLoader([]string{
		filepath.Join(baseDir, "world", "dnd-campaign"),
		filepath.Join(baseDir, "data"),
	})

	t.Run("Requires GM to start", func(t *testing.T) {
		params := map[string]any{"action": "start"}
		_, err := ExecuteGenericCommand("encounter", "Paulo", nil, params, "", state, loader, testReg(loader))
		if err == nil {
			t.Fatalf("Expected error when non-GM tries to start encounter")
		}
	})

	t.Run("End fails when no active encounter", func(t *testing.T) {
		params := map[string]any{"action": "end"}
		_, err := ExecuteGenericCommand("encounter", "GM", nil, params, "", state, loader, testReg(loader))
		if err == nil {
			t.Fatalf("Expected error when ending a non-existent encounter")
		}
	})

	t.Run("Start conflict when already active", func(t *testing.T) {
		state.IsEncounterActive = true // mock

		params := map[string]any{"action": "start"}
		_, err := ExecuteGenericCommand("encounter", "GM", nil, params, "", state, loader, testReg(loader))
		if err == nil {
			t.Fatalf("Expected error when starting encounter over an active one")
		}

		state.IsEncounterActive = false // reset
	})
}
