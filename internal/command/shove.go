package command

import (
	"fmt"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
	"github.com/suderio/dndsl/internal/rules"
)

// ExecuteShove handles the shove action with size restrictions and saving throws.
func ExecuteShove(cmd *parser.ActionCmd, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
	if state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	attacker, ok := state.Entities[currentActor]
	if !ok {
		return nil, fmt.Errorf("current actor %s not found", currentActor)
	}

	if attacker.ActionsRemaining <= 0 {
		return nil, fmt.Errorf("%s has no actions remaining this turn", currentActor)
	}

	target, ok := state.Entities[cmd.Target]
	if !ok {
		return nil, fmt.Errorf("target %s not found", cmd.Target)
	}

	// 1. Size Check
	attackerSize := data.SizeUnknown
	targetSize := data.SizeUnknown

	// Attacker Size
	if attacker.Category == "Monster" {
		mon, err := loader.LoadMonster(attacker.ID)
		if err == nil {
			attackerSize = data.ParseSize(mon.Size)
		}
	} else {
		char, err := loader.LoadCharacter(attacker.ID)
		if err == nil {
			if char.Race != "" {
				race, err := loader.LoadRace(char.Race)
				if err == nil {
					attackerSize = data.ParseSize(race.Size)
				}
			}
		}
	}

	// Target Size
	if target.Category == "Monster" {
		mon, err := loader.LoadMonster(target.ID)
		if err == nil {
			targetSize = data.ParseSize(mon.Size)
		}
	} else {
		char, err := loader.LoadCharacter(target.ID)
		if err == nil {
			if char.Race != "" {
				race, err := loader.LoadRace(char.Race)
				if err == nil {
					targetSize = data.ParseSize(race.Size)
				}
			}
		}
	}

	if !data.CanShove(attackerSize, targetSize) {
		return nil, fmt.Errorf("%s is too large for %s to shove", cmd.Target, currentActor)
	}

	// 2. DC Calculation
	dc := 10
	if attacker.Category == "Character" {
		char, _ := loader.LoadCharacter(attacker.ID)
		dc = 8 + data.CalculateModifier(char.Strength) + char.ProficiencyBonus
	} else {
		mon, _ := loader.LoadMonster(attacker.ID)
		dc = 8 + data.CalculateModifier(mon.Strength) + mon.ProficiencyBonus
	}

	// 3. Events
	msg := fmt.Sprintf("%s attempted to shove %s", currentActor, cmd.Target)
	return []engine.Event{
		&engine.ActionConsumedEvent{ActorID: currentActor},
		&engine.HintEvent{MessageStr: msg},
		&engine.AskIssuedEvent{
			Targets: []string{cmd.Target},
			Check:   []string{"strength", "save", "or", "dexterity", "save"},
			DC:      dc,
			// Fails: GM manually applies Prone or Push as per user request
		},
	}, nil
}
