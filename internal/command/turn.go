package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
	"github.com/suderio/dndsl/internal/rules"
)

// ExecuteTurn forces the initiative rotation explicitly
func ExecuteTurn(cmd *parser.TurnCmd, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
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

	if state.CurrentTurn < 0 {
		return nil, fmt.Errorf("combat has not started (roll initiative first)")
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	if !strings.EqualFold(actorName, "GM") && !strings.EqualFold(actorName, strings.ReplaceAll(currentActor, "-", "_")) && !strings.EqualFold(actorName, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	events := []engine.Event{
		&engine.TurnEndedEvent{ActorID: currentActor},
	}

	// 1. Reset actor stats (Actions, etc.)
	if ent, ok := state.Entities[currentActor]; ok {
		ent.ActionsRemaining = 1
		ent.BonusActionsRemaining = 1
		ent.ReactionsRemaining = 1
		ent.AttacksRemaining = 0

		// Clear Disengaged condition
		newConditions := []string{}
		for _, c := range ent.Conditions {
			if strings.ToLower(c) != "disengaged" {
				newConditions = append(newConditions, c)
			}
		}
		ent.Conditions = newConditions
	}

	// Determine next actor
	nextIndex := (state.CurrentTurn + 1) % len(state.TurnOrder)
	nextActor := state.TurnOrder[nextIndex]

	// Handle Monster Recharge Logic for the next actor
	if spent, ok := state.SpentRecharges[nextActor]; ok && len(spent) > 0 {
		mon, err := loader.LoadMonster(nextActor)
		if err == nil {
			for _, actionName := range spent {
				for _, a := range mon.Actions {
					if strings.EqualFold(a.Name, actionName) && a.Recharge != "" {
						// Roll d6
						res, _ := engine.Roll(&parser.DiceExpr{Raw: "1d6"})

						success := false
						if a.Recharge == "6" && res.Total == 6 {
							success = true
						} else if a.Recharge == "5-6" && res.Total >= 5 {
							success = true
						}

						events = append(events, &engine.RechargeRolledEvent{
							ActorID:     nextActor,
							ActionName:  a.Name,
							Roll:        res.Total,
							Requirement: a.Recharge,
							Success:     success,
						})

						if success {
							events = append(events, &engine.AbilityRechargedEvent{
								ActorID:    nextActor,
								ActionName: a.Name,
							})
						}
					}
				}
			}
		}
	}

	events = append(events, &engine.TurnChangedEvent{ActorID: nextActor})

	return events, nil
}
