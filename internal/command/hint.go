package command

import (
	"fmt"
	"strings"

	"dndsl/internal/engine"
	"dndsl/internal/parser"
)

// ExecuteHint analyzes the GameState and explains the current block or turn rotation
func ExecuteHint(cmd *parser.HintCmd, state *engine.GameState) ([]engine.Event, error) {
	if !state.IsEncounterActive {
		return []engine.Event{&engine.HintEvent{MessageStr: "No active encounter."}}, nil
	}

	var missingInitiatives []string
	for id, ent := range state.Entities {
		if _, ok := state.Initiatives[id]; !ok {
			missingInitiatives = append(missingInitiatives, ent.Name)
		}
	}

	if len(missingInitiatives) > 0 {
		names := strings.Join(missingInitiatives, ", ")
		if len(state.TurnOrder) == 0 {
			return []engine.Event{&engine.HintEvent{
				MessageStr: fmt.Sprintf("Encounter started. Waiting for initiative of %s.", names),
			}}, nil
		}

		currentActorName := state.TurnOrder[state.CurrentTurn]
		if ent, ok := state.Entities[currentActorName]; ok {
			currentActorName = ent.Name
		}

		return []engine.Event{&engine.HintEvent{
			MessageStr: fmt.Sprintf("It's %s turn. Waiting for initiative roll of %s.", currentActorName, names),
		}}, nil
	}

	if len(state.PendingChecks) > 0 {
		var waiting []string
		for id := range state.PendingChecks {
			nameStr := id
			if ent, ok := state.Entities[id]; ok {
				nameStr = ent.Name
			}
			waiting = append(waiting, nameStr)
		}

		names := strings.Join(waiting, ", ")

		if len(state.TurnOrder) == 0 {
			return []engine.Event{&engine.HintEvent{
				MessageStr: fmt.Sprintf("Waiting for check of %s.", names),
			}}, nil
		}
		currentActorName := state.TurnOrder[state.CurrentTurn]
		if ent, ok := state.Entities[currentActorName]; ok {
			currentActorName = ent.Name
		}
		return []engine.Event{&engine.HintEvent{
			MessageStr: fmt.Sprintf("It's %s turn. Waiting for check of %s.", currentActorName, names),
		}}, nil
	}

	if len(state.TurnOrder) == 0 {
		return []engine.Event{&engine.HintEvent{MessageStr: "Encounter is active but has no participants."}}, nil
	}

	currentActorName := state.TurnOrder[state.CurrentTurn]
	if ent, ok := state.Entities[currentActorName]; ok {
		currentActorName = ent.Name
	}

	if state.PendingDamage != nil {
		return []engine.Event{&engine.HintEvent{
			MessageStr: fmt.Sprintf("It's %s turn. Waiting for damage of last attack.", currentActorName),
		}}, nil
	}

	return []engine.Event{&engine.HintEvent{
		MessageStr: fmt.Sprintf("It's %s turn.", currentActorName),
	}}, nil
}
