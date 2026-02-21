package command

import (
	"fmt"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

// RollInitiative encapsulates all standard D&D 5e logic required to generate an initiative roll
// allowing it to be securely reused across different command contexts
func RollInitiative(actorName string, res TargetRes, manualOverride *int) *engine.InitiativeRolledEvent {
	if manualOverride != nil {
		return &engine.InitiativeRolledEvent{
			ActorName: actorName,
			Score:     *manualOverride,
			IsManual:  true,
		}
	}

	rollExpr := &parser.DiceExpr{Raw: fmt.Sprintf("1d20%+d", res.InitiativeMod)}
	rollRes, _ := engine.Roll(rollExpr)

	return &engine.InitiativeRolledEvent{
		ActorName: actorName,
		Score:     rollRes.Total,
		IsManual:  false,
		RawRolls:  rollRes.RawRolls,
		Kept:      rollRes.Kept,
		Dropped:   rollRes.Dropped,
		Modifier:  res.InitiativeMod,
	}
}

// ExecuteInitiative handles the `initiative :by <Actor> [Override]` syntax
func ExecuteInitiative(cmd *parser.InitiativeCmd, state *engine.GameState, loader *data.Loader) ([]engine.Event, error) {
	if !state.IsEncounterActive {
		return nil, fmt.Errorf("conflict: cannot roll initiative without an active encounter")
	}

	actorName := cmd.Actor.Name

	if _, exists := state.Entities[actorName]; !exists {
		return nil, fmt.Errorf("conflict: %s is not actively participating in this encounter", actorName)
	}

	res, err := CheckEntityLocally(actorName, loader)
	if err != nil {
		return nil, err
	}

	evt := RollInitiative(actorName, res, cmd.Value)
	return []engine.Event{evt}, nil
}
