package command

import (
	"fmt"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

// ExecuteEncounter handles the `encounter by: <Actor> start|end` syntax
func ExecuteEncounter(cmd *parser.EncounterCmd, state *engine.GameState, loader *data.Loader) ([]engine.Event, error) {
	if cmd.Actor == nil {
		cmd.Actor = &parser.ActorExpr{Name: "GM"}
	}

	if err := ValidateGM(cmd.Actor); err != nil {
		return nil, err
	}

	isStart := cmd.Action == "start"

	if isStart && state.IsEncounterActive {
		return nil, fmt.Errorf("conflict: an encounter is already active. End it first")
	}
	if !isStart && !state.IsEncounterActive {
		return nil, fmt.Errorf("conflict: no active encounter to end")
	}

	var events []engine.Event

	if isStart {
		events = append(events, &engine.EncounterStartedEvent{})

		// Process the `with: <targets>` list
		for _, target := range cmd.Targets {
			res, err := CheckEntityLocally(target, loader)
			if err != nil {
				return nil, err // Fail immediately
			}

			// All verified additions generate an ActorAddedEvent
			// For now, we mock max HP to 10 as we haven't written the full YAML character sheet unmarshaler
			events = append(events, &engine.ActorAddedEvent{
				ID:    target,
				Name:  res.Name,
				MaxHP: 10,
			})

			// Monsters automatically roll initiative
			if res.Type == "Monster" {
				events = append(events, RollInitiative(target, res, nil))
			}
		}
	} else {
		events = append(events, &engine.EncounterEndedEvent{})
	}

	return events, nil
}
