package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
	"github.com/suderio/dndsl/internal/rules"
)

type explicitDamage struct {
	DiceMacro string
	Type      string
}

// ExecuteDamage deducts hits resolving back entirely mechanically through GameState buffers
func ExecuteDamage(cmd *parser.DamageCmd, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
	if state.PendingDamage == nil || state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}

	if state.CurrentTurn < 0 {
		return nil, fmt.Errorf("combat has not started (roll initiative first)")
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
	var damageInstances []explicitDamage

	if len(cmd.Rolls) > 0 {
		for _, r := range cmd.Rolls {
			damageInstances = append(damageInstances, explicitDamage{
				DiceMacro: r.Dice.Raw,
				Type:      strings.ToLower(r.Type),
			})
		}
	} else {
		// Use loader to find weapon
		found := false
		if char, err := loader.LoadCharacter(currentActor); err == nil {
			for _, a := range char.Actions {
				if strings.EqualFold(a.Name, weaponToUse) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(weaponToUse)) {
					if len(a.Damage) > 0 {
						for _, dm := range a.Damage {
							damageInstances = append(damageInstances, explicitDamage{
								DiceMacro: dm.DamageDice,
								Type:      strings.ToLower(dm.DamageType.Index),
							})
						}
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
							for _, dm := range a.Damage {
								damageInstances = append(damageInstances, explicitDamage{
									DiceMacro: dm.DamageDice,
									Type:      strings.ToLower(dm.DamageType.Index),
								})
							}
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

	// Two-Weapon Fighting: don't add positive ability modifier to off-hand attack damage
	if state.PendingDamage.IsOffHand {
		for i, inst := range damageInstances {
			if strings.Contains(inst.DiceMacro, "+") {
				parts := strings.Split(inst.DiceMacro, "+")
				damageInstances[i].DiceMacro = strings.TrimSpace(parts[0])
			}
		}
	}

	events := []engine.Event{}

	for _, t := range validTargets {
		targetResistances := []string{}
		targetImmunities := []string{}
		targetVulnerabilities := []string{}

		if char, err := loader.LoadCharacter(t); err == nil {
			for _, def := range char.Defenses {
				targetResistances = append(targetResistances, def.Resistances...)
				targetImmunities = append(targetImmunities, def.Immunities...)
				targetVulnerabilities = append(targetVulnerabilities, def.Vulnerabilities...)
			}
		} else if mon, err := loader.LoadMonster(t); err == nil {
			for _, def := range mon.Defenses {
				targetResistances = append(targetResistances, def.Resistances...)
				targetImmunities = append(targetImmunities, def.Immunities...)
				targetVulnerabilities = append(targetVulnerabilities, def.Vulnerabilities...)
			}
		}

		totalDamageDealt := 0

		for _, inst := range damageInstances {
			res, err := engine.Roll(&parser.DiceExpr{Raw: inst.DiceMacro})
			if err != nil {
				return nil, err
			}

			// Compute multi
			multiplier := 1.0
			foundType := inst.Type
			if foundType != "" {
				for _, im := range targetImmunities {
					if strings.EqualFold(im, foundType) {
						multiplier = 0.0
						break
					}
				}
				if multiplier > 0 {
					for _, res := range targetResistances {
						if strings.EqualFold(res, foundType) {
							multiplier = 0.5
							break
						}
					}
				}
				if multiplier > 0 {
					for _, vul := range targetVulnerabilities {
						if strings.EqualFold(vul, foundType) {
							multiplier = 2.0
							break
						}
					}
				}
			}

			computedDmg := int(float64(res.Total) * multiplier)
			totalDamageDealt += computedDmg

			events = append(events, &engine.DiceRolledEvent{
				ActorName: currentActor,
				Total:     computedDmg,
				RawRolls:  res.RawRolls,
				Kept:      res.Kept,
				Dropped:   res.Dropped,
				Modifier:  res.Modifier,
			})
		}

		events = append(events, &engine.HPChangedEvent{
			ActorID: t,
			Amount:  -totalDamageDealt,
		})
	}

	return events, nil
}
