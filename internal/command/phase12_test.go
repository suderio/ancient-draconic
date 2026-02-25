package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/rules"
)

func TestContestStartedEvent(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Attacker"] = &engine.Entity{ID: "Attacker", Name: "Attacker"}
	state.Entities["Defender"] = &engine.Entity{ID: "Defender", Name: "Defender"}

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})
	reg, _ := rules.NewRegistry(nil, func(s string) int { return 10 }, nil)

	// Create mock manifest command
	m, _ := loader.LoadManifest()
	m.Commands["test_contest"] = data.CommandDefinition{
		Name: "test_contest",
		Steps: []data.CommandStep{
			{
				Name:    "contest",
				Formula: "{'attacker_id': actor.id, 'defender_id': target.id, 'attacker_roll': 15, 'defender_options': 'athletics', 'resolves_with': 'ApplyTest'}",
				Event:   "ContestStarted",
			},
		},
	}
	reg, _ = rules.NewRegistry(m, func(s string) int { return 10 }, nil)

	events, err := ExecuteGenericCommand("test_contest", "Attacker", []string{"Defender"}, nil, "", state, loader, reg)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.IsType(t, &engine.ContestStartedEvent{}, events[0])

	err = events[0].Apply(state)
	assert.NoError(t, err)

	pending := state.Metadata["pending_contests"].(map[string]any)["Defender"].(map[string]any)
	assert.Equal(t, "Attacker", pending["attacker"])
	assert.Equal(t, 15, pending["attacker_roll"])
}

func TestChoiceIssuedEvent(t *testing.T) {
	state := engine.NewGameState()
	state.IsEncounterActive = true
	state.Entities["Target"] = &engine.Entity{ID: "Target", Name: "Target"}

	loader := data.NewLoader([]string{"../../world/dnd-campaign", "../../data"})
	reg, _ := rules.NewRegistry(nil, func(s string) int { return 10 }, nil)

	m, _ := loader.LoadManifest()
	m.Commands["test_choice"] = data.CommandDefinition{
		Name: "test_choice",
		Steps: []data.CommandStep{
			{
				Name:    "choice",
				Formula: "{'prompt': 'Pick a color', 'options': ['red', 'blue'], 'resolves_with': 'ApplyColor'}",
				Event:   "ChoiceIssued",
			},
		},
	}
	reg, _ = rules.NewRegistry(m, func(s string) int { return 10 }, nil)

	events, err := ExecuteGenericCommand("test_choice", "Target", []string{"Target"}, nil, "", state, loader, reg)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.IsType(t, &engine.ChoiceIssuedEvent{}, events[0])

	err = events[0].Apply(state)
	assert.NoError(t, err)

	pending := state.Metadata["pending_choices"].(map[string]any)["Target"].(map[string]any)
	assert.Equal(t, "Pick a color", pending["prompt"])
}
