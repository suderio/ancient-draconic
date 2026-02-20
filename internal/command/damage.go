package command

import (
	"fmt"
	"strings"

	"dndsl/internal/data"
	"dndsl/internal/engine"
	"dndsl/internal/parser"
)

// ExecuteDamage deducts hits resolving back entirely mechanically through GameState buffers
func ExecuteDamage(cmd *parser.DamageCmd, state *engine.GameState, loader *data.Loader) ([]engine.Event, error) {
	if state.PendingDamage == nil || state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	if !strings.EqualFold(actorName, "GM") && !strings.EqualFold(actorName, strings.ReplaceAll(currentActor, "-", "_")) && !strings.EqualFold(actorName, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	if !strings.EqualFold(state.PendingDamage.Attacker, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	// Filter targets that were actually hit
	var validTargets []string
	for _, t := range state.PendingDamage.Targets {
		if state.PendingDamage.HitStatus[t] {
			validTargets = append(validTargets, t)
		}
	}

	if len(validTargets) == 0 {
		return nil, engine.ErrSilentIgnore
	}

	weaponToUse := state.PendingDamage.Weapon
	if cmd.Weapon != "" {
		weaponToUse = cmd.Weapon
	}

	// Figure out damage roll
	var damageMacro string
	if cmd.Dice != nil {
		damageMacro = cmd.Dice.Raw
	} else {
		// Use loader to find weapon
		found := false
		if char, err := loader.LoadCharacter(currentActor); err == nil {
			for _, a := range char.Actions {
				if strings.EqualFold(a.Name, weaponToUse) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(weaponToUse)) {
					if len(a.Damage) > 0 {
						damageMacro = a.Damage[0].DamageDice
						found = true
						break
					}
				}
			}
		}
		if !found {
			if mon, err := loader.LoadMonster(currentActor); err == nil {
				for _, a := range mon.Actions {
					if strings.EqualFold(a.Name, weaponToUse) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(weaponToUse)) {
						if len(a.Damage) > 0 {
							damageMacro = a.Damage[0].DamageDice
							found = true
							break
						}
					}
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("could not find damage dice for weapon %s", weaponToUse)
		}
	}

	events := []engine.Event{}

	for _, t := range validTargets {
		res, err := engine.Roll(&parser.DiceExpr{Raw: damageMacro})
		if err != nil {
			return nil, err
		}
		events = append(events, &engine.DiceRolledEvent{
			ActorName: currentActor,
			Total:     res.Total,
			RawRolls:  res.RawRolls,
			Kept:      res.Kept,
			Dropped:   res.Dropped,
			Modifier:  res.Modifier,
		})
		events = append(events, &engine.HPChangedEvent{
			ActorID: t,
			Amount:  -res.Total,
		})
	}

	return events, nil
}
