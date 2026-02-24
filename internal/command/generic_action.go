package command

import (
	"fmt"
	"strings"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

// ExecuteAction handles standard 5e actions like Dash, Disengage, etc.
func ExecuteAction(cmd *parser.ActionCmd, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
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

	// In the generic model, we check ent.Spent["actions"]
	if ent.Spent["actions"] > 0 {
		return nil, fmt.Errorf("%s has no actions remaining this turn", currentActor)
	}

	actionType := strings.ToLower(cmd.Action)

	// Complex actions that might need adjudication
	if actionType == "shove" || actionType == "magic" || actionType == "ready" {
		pendingAdj, ok := state.Metadata["pending_adjudication"].(map[string]any)
		if !ok {
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

		if approved, ok := pendingAdj["approved"].(bool); !ok || !approved {
			return nil, fmt.Errorf("action is still pending GM adjudication")
		}
	}
	if actionType == "escape" {
		// Try to find who is grappling the current actor
		grapplerID := ""
		conditionToRemove := "grappled"
		for _, c := range ent.Conditions {
			if strings.HasPrefix(strings.ToLower(c), "grappledby:") {
				parts := strings.Split(c, ":")
				if len(parts) == 2 {
					grapplerID = parts[1]
					conditionToRemove = c
					break
				}
			}
		}

		dc := 10 // Baseline fallback
		if grapplerID != "" {
			// Calculate DC from grappler's stats
			char, err := loader.LoadCharacter(grapplerID)
			if err == nil {
				dc = 8 + data.CalculateModifier(char.Strength) + char.ProficiencyBonus
			} else {
				mon, err := loader.LoadMonster(grapplerID)
				if err == nil {
					dc = 8 + data.CalculateModifier(mon.Strength) + mon.ProficiencyBonus
				}
			}
		}

		return []engine.Event{
			&engine.ActionConsumedEvent{ActorID: currentActor},
			&engine.AskIssuedEvent{
				Targets: []string{currentActor},
				Check:   []string{"athletics", "or", "acrobatics"},
				DC:      dc,
				Succeeds: map[string]any{
					"remove_condition": conditionToRemove,
				},
			},
		}, nil
	}

	// Handle specific action logic
	var events []engine.Event
	events = append(events, &engine.ActionConsumedEvent{ActorID: currentActor})

	switch strings.ToLower(actionType) {
	case "disengage":
		events = append(events, &engine.ConditionAppliedEvent{
			ActorID:   currentActor,
			Condition: "Disengaged",
		})
	case "dash", "hide", "improvise", "influence", "ready", "search", "study", "utilize":
		// Simple action consumption + log
	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}

	// Logging event for the action
	msg := fmt.Sprintf("%s used %s", currentActor, cmd.Action)
	if cmd.Target != "" {
		msg += " on " + cmd.Target
	}
	events = append(events, &engine.HintEvent{MessageStr: msg})

	return events, nil
}
