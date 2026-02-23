package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
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

	if state.CurrentTurn < 0 {
		return nil, fmt.Errorf("combat has not started (roll initiative first)")
	}

	currentTurnActor := state.TurnOrder[state.CurrentTurn]
	actingActor := currentTurnActor
	if cmd.Actor != nil {
		actingActor = cmd.Actor.Name
	}

	// Turn order enforcement:
	// - Reactions can be taken by anyone at any time (if they have reactions remaining).
	// - Actions and Bonus Actions can only be taken by the active actor or GM.
	isGM := strings.EqualFold(actorName, "GM")
	isActiveActor := strings.EqualFold(actingActor, currentTurnActor) || strings.EqualFold(actingActor, strings.ReplaceAll(currentTurnActor, "-", "_"))

	if !cmd.Opportunity && !isGM && !isActiveActor {
		return nil, engine.ErrSilentIgnore
	}

	ent, ok := state.Entities[actingActor]
	if !ok {
		return nil, fmt.Errorf("actor %s not found in encounter", actingActor)
	}

	// Handle Opportunity Attack Adjudication
	if cmd.Opportunity {
		if ent.ReactionsRemaining <= 0 {
			return nil, fmt.Errorf("%s has no reactions remaining", actingActor)
		}
		if state.PendingAdjudication == nil {
			originalCmd := fmt.Sprintf("opportunity attack by: %s with: %s to: %s", actingActor, cmd.Weapon, strings.Join(cmd.Targets, " and "))
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

	// Enforce action economy for Actions and Bonus Actions
	// Enforce action economy for Actions and Off-hand Attacks
	if cmd.OffHand {
		if ent.BonusActionsRemaining <= 0 {
			return nil, fmt.Errorf("%s has no bonus actions remaining", actingActor)
		}
		if !ent.HasAttackedThisTurn {
			return nil, fmt.Errorf("%s must take the Attack action before taking an off-hand attack", actingActor)
		}
		if strings.EqualFold(cmd.Weapon, ent.LastAttackedWithWeapon) || strings.Contains(strings.ToLower(ent.LastAttackedWithWeapon), strings.ToLower(cmd.Weapon)) {
			return nil, fmt.Errorf("off-hand attack must use a different weapon than the main attack (%s)", ent.LastAttackedWithWeapon)
		}
	} else if !cmd.Opportunity {
		if ent.ActionsRemaining <= 0 && ent.AttacksRemaining <= 0 {
			return nil, fmt.Errorf("%s has no actions or attacks remaining this turn", actingActor)
		}
	}

	// Check if this specific command exceeds remaining multi-attacks (only for standard actions)
	neededAttacks := len(cmd.Targets)
	availableAttacks := 1 // default for off-hand/opportunity
	if !cmd.OffHand && !cmd.Opportunity {
		availableAttacks = ent.AttacksRemaining
		if ent.ActionsRemaining > 0 && availableAttacks <= 0 {
			availableAttacks = 1
		}
	}

	if neededAttacks > availableAttacks {
		return nil, fmt.Errorf("%s only has %d attack(s) remaining, but tried to target %d", actingActor, availableAttacks, neededAttacks)
	}

	// Try resolving the physical attacker in game memory
	var attackBonus int
	var recharge string
	var resolvedWeaponName string
	attackerFound := false

	// Default matching for attacker sheet (Characters or Monsters map weapons)
	if char, err := loader.LoadCharacter(actingActor); err == nil {
		for _, a := range char.Actions {
			if strings.EqualFold(a.Name, cmd.Weapon) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(cmd.Weapon)) {
				attackBonus = a.AttackBonus
				recharge = a.Recharge
				resolvedWeaponName = a.Name
				attackerFound = true
				break
			}
		}
	}
	if !attackerFound {
		if mon, err := loader.LoadMonster(actingActor); err == nil {
			for _, a := range mon.Actions {
				if strings.EqualFold(a.Name, cmd.Weapon) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(cmd.Weapon)) {
					attackBonus = a.AttackBonus
					recharge = a.Recharge
					resolvedWeaponName = a.Name
					attackerFound = true
					break
				}
			}
		}
	}

	if attackerFound {
		// Check if the resolved weapon is currently spent
		if spent, ok := state.SpentRecharges[actingActor]; ok {
			for _, s := range spent {
				if strings.EqualFold(s, resolvedWeaponName) {
					return nil, fmt.Errorf("ability %s is still cooling down", resolvedWeaponName)
				}
			}
		}
	}

	if !attackerFound && cmd.Dice == nil {
		return nil, fmt.Errorf("attacker %s does not have weapon %s and no override dice was given", actingActor, cmd.Weapon)
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

	finalWeaponName := cmd.Weapon
	if resolvedWeaponName != "" {
		finalWeaponName = resolvedWeaponName
	}

	events = append(events, &engine.AttackResolvedEvent{
		Attacker:      actingActor, // Maps to the actor whose action it actually is
		Weapon:        finalWeaponName,
		Targets:       cmd.Targets,
		HitStatus:     hitStatus,
		IsOffHand:     cmd.OffHand,
		IsOpportunity: cmd.Opportunity,
	})

	// Check for and consume Help benefit on the targets
	for _, targetName := range cmd.Targets {
		if target, ok := state.Entities[targetName]; ok {
			for _, c := range target.Conditions {
				if strings.HasPrefix(c, "HelpedAttack:") {
					events = append(events, &engine.ConditionRemovedEvent{
						ActorID:   targetName,
						Condition: c,
					})
					break // Consume only one distraction per target per attack action
				}
			}
		}
	}

	if recharge != "" {
		events = append(events, &engine.AbilitySpentEvent{
			ActorID:    actingActor,
			ActionName: resolvedWeaponName,
		})
	}

	return events, nil
}
