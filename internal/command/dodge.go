package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
	"github.com/suderio/dndsl/internal/rules"
)

// ExecuteDodge handles the `dodge by: <actor>` command
func ExecuteDodge(cmd *parser.DodgeCmd, state *engine.GameState, reg *rules.Registry) ([]engine.Event, error) {
	if state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}

	if len(state.TurnOrder) == 0 || state.CurrentTurn < 0 {
		return nil, fmt.Errorf("combat has not started")
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	// Basic turn order enforcement: only the active actor or GM can initiate
	if !strings.EqualFold(actorName, "GM") && !strings.EqualFold(actorName, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	ent, ok := state.Entities[currentActor]
	if !ok {
		return nil, fmt.Errorf("actor %s not found in encounter", currentActor)
	}

	if ent.ActionsRemaining <= 0 {
		return nil, fmt.Errorf("%s has no actions remaining this turn", currentActor)
	}

	return []engine.Event{
		&engine.DodgeTakenEvent{
			ActorID: currentActor,
		},
	}, nil
}
