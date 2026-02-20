package command

import (
	"fmt"
	"strings"

	"dndsl/internal/data"
	"dndsl/internal/engine"
	"dndsl/internal/parser"
)

// ExecuteAttack resolves "meets it, beats it" math recursively against target Arrays
func ExecuteAttack(cmd *parser.AttackCmd, state *engine.GameState, loader *data.Loader) ([]engine.Event, error) {
	if state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}

	if len(state.TurnOrder) == 0 {
		return nil, fmt.Errorf("no active encounter to attack in")
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	if !strings.EqualFold(actorName, "GM") && !strings.EqualFold(actorName, strings.ReplaceAll(currentActor, "-", "_")) && !strings.EqualFold(actorName, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	// Try resolving the physical attacker in game memory
	var attackBonus int
	attackerFound := false

	// Default matching for attacker sheet (Characters or Monsters map weapons)
	if char, err := loader.LoadCharacter(currentActor); err == nil {
		for _, a := range char.Actions {
			if strings.EqualFold(a.Name, cmd.Weapon) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(cmd.Weapon)) {
				attackBonus = a.AttackBonus
				attackerFound = true
				break
			}
		}
	}
	if !attackerFound {
		if mon, err := loader.LoadMonster(currentActor); err == nil {
			for _, a := range mon.Actions {
				if strings.EqualFold(a.Name, cmd.Weapon) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(cmd.Weapon)) {
					attackBonus = a.AttackBonus
					attackerFound = true
					break
				}
			}
		}
	}

	if !attackerFound && cmd.Dice == nil {
		return nil, fmt.Errorf("attacker %s does not have weapon %s and no override dice was given", currentActor, cmd.Weapon)
	}

	events := []engine.Event{}
	hitStatus := make(map[string]bool)

	for _, targetName := range cmd.Targets {
		// Clean up parser syntax (some names map with dashes differently)
		cleanTarget := targetName

		// Find Target AC
		var targetAC int
		acFound := false
		if char, err := loader.LoadCharacter(cleanTarget); err == nil {
			if len(char.ArmorClass) > 0 {
				targetAC = char.ArmorClass[0].Value
				acFound = true
			}
		}
		if !acFound {
			if mon, err := loader.LoadMonster(cleanTarget); err == nil {
				if len(mon.ArmorClass) > 0 {
					targetAC = mon.ArmorClass[0].Value
					acFound = true
				}
			}
		}

		if !acFound {
			// fallback directly to GameState if not in config schema
			if _, ok := state.Entities[cleanTarget]; ok {
				targetAC = 10
				acFound = true
			}
		}

		if !acFound {
			return nil, fmt.Errorf("could not resolve AC for target %s", targetName)
		}

		// Roll or Override Attack
		total := 0
		if cmd.Dice != nil {
			// Override
			res, err := engine.Roll(cmd.Dice)
			if err != nil {
				return nil, err
			}
			events = append(events, &engine.DiceRolledEvent{
				ActorName: actorName,
				Total:     res.Total,
				RawRolls:  res.RawRolls,
				Kept:      res.Kept,
				Dropped:   res.Dropped,
				Modifier:  res.Modifier,
			})
			total = res.Total
		} else {
			// Auto calculate
			hasAdv, hasDis := GetConditionMatrixForAttack(actorName, cleanTarget, state)
			baseDice := "1d20"
			if hasAdv && !hasDis {
				baseDice = "2d20kh1"
			} else if hasDis && !hasAdv {
				baseDice = "2d20kl1"
			}

			modSuffix := ""
			if attackBonus >= 0 {
				modSuffix = fmt.Sprintf("+%d", attackBonus)
			} else {
				modSuffix = fmt.Sprintf("%d", attackBonus)
			}
			diceStr := fmt.Sprintf("%s%s", baseDice, modSuffix)
			res, err := engine.Roll(&parser.DiceExpr{Raw: diceStr})
			if err != nil {
				return nil, err
			}
			events = append(events, &engine.DiceRolledEvent{
				ActorName: actorName,
				Total:     res.Total,
				RawRolls:  res.RawRolls,
				Kept:      res.Kept,
				Dropped:   res.Dropped,
				Modifier:  res.Modifier,
			})
			total = res.Total
		}

		hitStatus[targetName] = total >= targetAC
	}

	events = append(events, &engine.AttackResolvedEvent{
		Attacker:  currentActor, // Maps to the actor whose turn it actually is
		Weapon:    cmd.Weapon,
		Targets:   cmd.Targets,
		HitStatus: hitStatus,
	})

	return events, nil
}
