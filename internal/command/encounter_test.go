package command_test

import (
	"path/filepath"
	"testing"

	"github.com/suderio/ancient-draconic/internal/command"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
)

func TestExecuteEncounterStateValidations(t *testing.T) {
	state := engine.NewGameState()
	baseDir := filepath.Join("..", "..") // Assume root is two levels up from command/ for test files
	loader := data.NewLoader([]string{filepath.Join(baseDir, "data")})

	t.Run("Requires GM to start", func(t *testing.T) {
		cmd := &parser.EncounterCmd{
			Action: "start",
			Actor:  &parser.ActorExpr{Name: "Paulo"},
		}

		_, err := command.ExecuteEncounter(cmd, state, loader, nil)
		if err == nil {
			t.Fatalf("Expected error when non-GM tries to start encounter")
		}
	})

	t.Run("End fails when no active encounter", func(t *testing.T) {
		cmd := &parser.EncounterCmd{
			Action: "end",
			Actor:  &parser.ActorExpr{Name: "GM"},
		}

		_, err := command.ExecuteEncounter(cmd, state, loader, nil)
		if err == nil {
			t.Fatalf("Expected error when ending a non-existent encounter")
		}
	})

	t.Run("Start conflict when already active", func(t *testing.T) {
		state.IsEncounterActive = true // mock

		cmd := &parser.EncounterCmd{
			Action: "start",
			Actor:  &parser.ActorExpr{Name: "GM"},
		}

		_, err := command.ExecuteEncounter(cmd, state, loader, nil)
		if err == nil {
			t.Fatalf("Expected error when starting encounter over an active one")
		}

		state.IsEncounterActive = false // reset
	})
}
