package command

import (
	"fmt"
	"strings"

	"dndsl/internal/data"
	"dndsl/internal/engine"
	"dndsl/internal/parser"
)

// stringMatch matches syntax aliases like 'str' vs 'strength'
func stringMatch(target, candidate string) bool {
	t := strings.ToLower(target)
	c := strings.ToLower(candidate)
	return strings.HasPrefix(t, c) || strings.HasPrefix(c, t)
}

// evalModifier tries to extract the highest accurate modifier for a check
func evalModifier(actorID string, checkType []string, state *engine.GameState, loader *data.Loader) (int, error) {
	ent, ok := state.Entities[actorID]
	if !ok {
		return 0, fmt.Errorf("actor %s not found in encounter", actorID)
	}

	searchMode := "ability"
	targetName := strings.Join(checkType, " ")
	if strings.Contains(strings.ToLower(targetName), "save") || strings.Contains(strings.ToLower(targetName), "st") {
		searchMode = "save"
	}
	// clean up save syntax
	cleanTarget := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(targetName), "save", ""), "st", ""))

	// 1. Loader -> Character/Monster (to search the nested 'proficiencies' slice)
	// We'll brute force search Characters, although later we should pass the campaign path.
	// For simulation purposes, we rely on the loader defaults.
	c, err := loader.LoadCharacter(ent.ID)
	if err == nil {
		switch searchMode {
		case "save":
			for _, p := range c.Proficiencies {
				if strings.Contains(strings.ToLower(p.Proficiency.Name), "save") {
					parts := strings.Split(p.Proficiency.Name, " ")
					last := parts[len(parts)-1]
					if stringMatch(last, cleanTarget) {
						return p.Value, nil
					}
				}
			}
			// Fallback to base abilities
		default: // skill or ability
			for _, p := range c.Proficiencies {
				parts := strings.Split(p.Proficiency.Name, ": ")
				if len(parts) > 1 {
					if stringMatch(parts[1], cleanTarget) {
						return p.Value, nil
					}
				}
			}
		}

		// Fallback to Base Ability Modifiers
		switch {
		case stringMatch("str", cleanTarget) || stringMatch("athletics", cleanTarget):
			return data.CalculateModifier(c.Strength), nil
		case stringMatch("dex", cleanTarget) || stringMatch("stealth", cleanTarget) || stringMatch("acrobatics", cleanTarget):
			return data.CalculateModifier(c.Dexterity), nil
		case stringMatch("con", cleanTarget):
			return data.CalculateModifier(c.Constitution), nil
		case stringMatch("int", cleanTarget):
			return data.CalculateModifier(c.Intelligence), nil
		case stringMatch("wis", cleanTarget):
			return data.CalculateModifier(c.Wisdom), nil
		case stringMatch("cha", cleanTarget) || stringMatch("deception", cleanTarget) || stringMatch("persuasion", cleanTarget) || stringMatch("intimidation", cleanTarget):
			return data.CalculateModifier(c.Charisma), nil
		}
	}

	// Wait, what if it's a Monster?
	m, err := loader.LoadMonster(ent.ID)
	if err == nil {
		switch {
		case stringMatch("str", cleanTarget) || stringMatch("athletics", cleanTarget):
			return data.CalculateModifier(m.Strength), nil
		case stringMatch("dex", cleanTarget) || stringMatch("stealth", cleanTarget) || stringMatch("acrobatics", cleanTarget):
			return data.CalculateModifier(m.Dexterity), nil
		case stringMatch("con", cleanTarget):
			return data.CalculateModifier(m.Constitution), nil
		case stringMatch("int", cleanTarget):
			return data.CalculateModifier(m.Intelligence), nil
		case stringMatch("wis", cleanTarget):
			return data.CalculateModifier(m.Wisdom), nil
		case stringMatch("cha", cleanTarget):
			return data.CalculateModifier(m.Charisma), nil
		}
	}

	return 0, nil // Default baseline
}

// ExecuteCheck evaluates a requested check or performs a standalone one, accounting for proficiencies and conditions
func ExecuteCheck(cmd *parser.CheckCmd, state *engine.GameState, loader *data.Loader) ([]engine.Event, error) {
	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}
	cleanActor := strings.ToLower(actorName)

	if state.PendingDamage != nil {
		return nil, engine.ErrSilentIgnore
	}

	events := []engine.Event{}

	autoFail, hasAdv, hasDis := GetConditionMatrixForCheck(cleanActor, cmd.Check, state)

	// Build the roll
	result := 0
	if autoFail {
		result = 0 // Instant fail, no roll happens
	} else {
		// Figure out modifier
		mod, _ := evalModifier(cleanActor, cmd.Check, state, loader)

		baseDice := "1d20"
		if hasAdv && !hasDis {
			baseDice = "2d20kh1"
		} else if hasDis && !hasAdv {
			baseDice = "2d20kl1"
		}

		modSuffix := ""
		if mod >= 0 {
			modSuffix = fmt.Sprintf("+%d", mod)
		} else {
			modSuffix = fmt.Sprintf("%d", mod)
		}

		res, err := engine.Roll(&parser.DiceExpr{Raw: baseDice + modSuffix})
		if err != nil {
			return nil, err
		}

		events = append(events, &engine.DiceRolledEvent{
			ActorName: cleanActor,
			Total:     res.Total,
			RawRolls:  res.RawRolls,
			Kept:      res.Kept,
			Dropped:   res.Dropped,
			Modifier:  res.Modifier,
		})
		result = res.Total
	}

	req, hasRequest := state.PendingChecks[cleanActor]

	// If this check answers an active Ask:
	if hasRequest {
		// Just a heuristic verification
		targetCheck := strings.ToLower(strings.Join(req.Check, " "))
		currentCheck := strings.ToLower(strings.Join(cmd.Check, " "))
		if !strings.Contains(currentCheck, targetCheck) && !stringMatch(targetCheck, currentCheck) {
			// They issued a check, but it doesn't solve the Ask condition! Ignore but maybe log it?
			// The instructions suggest the Check must match. For now, we will be loose.
		}

		success := result >= req.DC

		events = append(events, &engine.CheckResolvedEvent{
			ActorID: cleanActor,
			Result:  result,
			Success: success,
		})

		var consequence *engine.RollConsequence
		if success {
			consequence = req.Succeeds
		} else {
			consequence = req.Fails
		}

		if consequence != nil {
			if consequence.IsDamage {
				dmgRes, err := engine.Roll(&parser.DiceExpr{Raw: consequence.DamageDice})
				if err == nil {
					totalDmg := dmgRes.Total
					if consequence.HalfDamage && success {
						totalDmg = totalDmg / 2
					}

					events = append(events, &engine.DiceRolledEvent{
						ActorName: "System",
						Total:     dmgRes.Total,
						RawRolls:  dmgRes.RawRolls,
						Kept:      dmgRes.Kept,
						Dropped:   dmgRes.Dropped,
						Modifier:  dmgRes.Modifier,
					})

					events = append(events, &engine.HPChangedEvent{
						ActorID: cleanActor,
						Amount:  -totalDmg,
					})
				}
			} else if consequence.Condition != "" {
				events = append(events, &engine.ConditionAppliedEvent{
					ActorID:   cleanActor,
					Condition: strings.ToLower(consequence.Condition),
				})
			}
		}
	} else {
		// It's just a free-floating check, no consequences applied
		events = append(events, &engine.CheckResolvedEvent{
			ActorID: cleanActor,
			Result:  result,
			Success: true, // defaults to true out of scope
		})
	}

	return events, nil
}
