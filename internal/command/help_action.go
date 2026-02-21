package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

// ExecuteHelpAction handles the mechanical help action
func ExecuteHelpAction(cmd *parser.HelpActionCmd, state *engine.GameState) ([]engine.Event, error) {
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

	// Mechanical help ALWAYS requires adjudication as per user requirements for complex actions
	if state.PendingAdjudication == nil {
		originalCmd := fmt.Sprintf("help by: %s %s to: %s", currentActor, cmd.Type, cmd.Target)
		return []engine.Event{
			&engine.AdjudicationStartedEvent{
				OriginalCommand: originalCmd,
			},
		}, nil
	}

	if !state.PendingAdjudication.Approved {
		return nil, fmt.Errorf("action is still pending GM adjudication")
	}

	// If approved, trigger the benefit
	return []engine.Event{
		&engine.HelpTakenEvent{
			HelperID: currentActor,
			TargetID: cmd.Target,
			HelpType: strings.ToLower(cmd.Type),
		},
	}, nil
}
