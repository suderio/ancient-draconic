package command

import (
	"fmt"
	"strings"

	"dndsl/internal/engine"
	"dndsl/internal/parser"
)

// ExecuteTurn forces the initiative rotation explicitly
func ExecuteTurn(cmd *parser.TurnCmd, state *engine.GameState) ([]engine.Event, error) {
	if state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}

	// Validate actor has the right to end turn
	if len(state.TurnOrder) == 0 {
		return nil, fmt.Errorf("no active encounter to end turn")
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	if !strings.EqualFold(actorName, "GM") && !strings.EqualFold(actorName, strings.ReplaceAll(currentActor, "-", "_")) && !strings.EqualFold(actorName, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	return []engine.Event{
		&engine.TurnEndedEvent{ActorID: currentActor},
	}, nil
}
