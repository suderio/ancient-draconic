package command

import (
	"fmt"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

// ExecuteAdd handles the `add by: GM <Name> and: <Name>` syntax
func ExecuteAdd(cmd *parser.AddCmd, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
	if cmd.Actor == nil {
		cmd.Actor = &parser.ActorExpr{Name: "GM"}
	}

	if err := ValidateGM(cmd.Actor); err != nil {
		return nil, err
	}

	if !state.IsEncounterActive {
		return nil, fmt.Errorf("conflict: cannot add actors without an active encounter")
	}

	var events []engine.Event
	for _, target := range cmd.Targets {
		res, err := CheckEntityLocally(target, loader)
		if err != nil {
			return nil, err // Fail immediately
		}

		if _, exists := state.Entities[target]; exists {
			return nil, fmt.Errorf("conflict: actor %s is already in the encounter", target)
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
			Defenses:   res.Defenses,
		})

		// Monsters automatically roll initiative
		if res.Category == "Monster" {
			events = append(events, RollInitiative(target, res, nil))
		}
	}

	return events, nil
}
