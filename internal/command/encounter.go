package command

import (
	"fmt"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
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
			events = append(events, &engine.ActorAddedEvent{
				ID:         target,
				Category:   res.Category,
				EntityType: res.EntityType,
				Name:       res.Name,
				MaxHP:      res.HP,
				Stats:      res.Stats,
				Abilities:  res.Abilities,
			})

			// Monsters automatically roll initiative
			if res.Category == "Monster" {
				events = append(events, RollInitiative(target, res, nil))
			}
		}
	} else {
		events = append(events, &engine.EncounterEndedEvent{})
	}

	return events, nil
}
