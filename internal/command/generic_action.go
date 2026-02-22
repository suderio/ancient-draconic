package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

// ExecuteAction handles standard 5e actions like Dash, Disengage, etc.
func ExecuteAction(cmd *parser.ActionCmd, state *engine.GameState) ([]engine.Event, error) {
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

	actionType := strings.ToLower(cmd.Action)

	// Complex actions that might need adjudication
	if actionType == "shove" || actionType == "magic" || actionType == "ready" {
		if state.PendingAdjudication == nil {
			originalCmd := fmt.Sprintf("%s by: %s", actionType, currentActor)
			if cmd.Target != "" {
				originalCmd += " to: " + cmd.Target
			}
			return []engine.Event{
				&engine.AdjudicationStartedEvent{
					OriginalCommand: originalCmd,
				},
			}, nil
		}

		if !state.PendingAdjudication.Approved {
			return nil, fmt.Errorf("action is still pending GM adjudication")
		}
	}
	if actionType == "escape" {
		return []engine.Event{
			&engine.ActionConsumedEvent{ActorID: currentActor},
			&engine.AskIssuedEvent{
				Targets: []string{currentActor},
				Check:   []string{"athletics", "or", "acrobatics"},
				DC:      10, // Default baseline, GM can override or adjudicate
				Succeeds: &engine.RollConsequence{
					RemoveCondition: "grappled",
				},
			},
		}, nil
	}

	// Logging event for the action
	msg := fmt.Sprintf("%s used %s", currentActor, cmd.Action)
	if cmd.Target != "" {
		msg += " on " + cmd.Target
	}

	return []engine.Event{
		&engine.ActionConsumedEvent{
			ActorID: currentActor,
		},
		&engine.HintEvent{
			MessageStr: msg,
		},
	}, nil
}
