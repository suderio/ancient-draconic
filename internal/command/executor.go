package command

import (
	"fmt"
	"strings"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

const (
	ActorGM = "GM"
)

// EntityAction represents resolved action data from a character/monster sheet
type EntityAction struct {
	Name        string
	AttackBonus int
	HitRule     string
	Recharge    string
	DamageDice  string
	DamageType  string
}

// ResolveEntityAction looks up a named action in the actor's data files (character sheet or monster data).
// It performs a case-insensitive search and supports partial matches for convenience.
func ResolveEntityAction(actorID string, actionName string, loader *data.Loader) (*EntityAction, bool) {
	if actionName == "" {
		return nil, false
	}
	// Try character first
	if char, err := loader.LoadCharacter(actorID); err == nil {
		for _, a := range char.Actions {
			if strings.EqualFold(a.Name, actionName) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(actionName)) {
				res := &EntityAction{
					Name:        a.Name,
					AttackBonus: a.AttackBonus,
					HitRule:     a.HitRule,
					Recharge:    a.Recharge,
				}
				if len(a.Damage) > 0 {
					res.DamageDice = a.Damage[0].DamageDice
					res.DamageType = a.Damage[0].DamageType.Index
				}
				return res, true
			}
		}
	}
	// Try monster
	if mon, err := loader.LoadMonster(actorID); err == nil {
		for _, a := range mon.Actions {
			if strings.EqualFold(a.Name, actionName) || strings.Contains(strings.ToLower(a.Name), strings.ToLower(actionName)) {
				res := &EntityAction{
					Name:        a.Name,
					AttackBonus: a.AttackBonus,
					HitRule:     a.HitRule,
					Recharge:    a.Recharge,
				}
				if len(a.Damage) > 0 {
					res.DamageDice = a.Damage[0].DamageDice
					res.DamageType = a.Damage[0].DamageType.Index
				}
				return res, true
			}
		}
	}
	return nil, false
}

// ResolveActor determines which entity is taking the action, accounting for turn order and overrides.
func ResolveActor(actorExpr *parser.ActorExpr, state *engine.GameState) (string, error) {
	if state.IsFrozen() {
		return "", engine.ErrSilentIgnore
	}

	actorName := ActorGM
	if actorExpr != nil {
		actorName = actorExpr.Name
	}

	if len(state.TurnOrder) == 0 {
		return "", fmt.Errorf("no active encounter to take actions in")
	}

	if state.CurrentTurn < 0 {
		return "", fmt.Errorf("combat has not started (roll initiative first)")
	}

	currentTurnActor := state.TurnOrder[state.CurrentTurn]
	actingActor := currentTurnActor
	if actorExpr != nil {
		actingActor = actorExpr.Name
	}

	// Turn order enforcement (legacy but shared for now)
	isGM := strings.EqualFold(actorName, ActorGM)
	isActiveActor := strings.EqualFold(actingActor, currentTurnActor) || strings.EqualFold(actingActor, strings.ReplaceAll(currentTurnActor, "-", "_"))

	if !isGM && !isActiveActor {
		return "", engine.ErrSilentIgnore
	}

	if _, ok := state.Entities[actingActor]; !ok {
		return "", fmt.Errorf("actor '%s' not found in encounter", actingActor)
	}

	return actingActor, nil
}

// ExecuteGenericCommand is the centralized entry point for manifest-driven logic.
// It maps engine states and parameters to a CEL context and evaluates the manifest definition.
// ExecuteGenericCommand is the central engine dispatcher for manifest-driven commands.
// It resolves the actor, normalizes parameters, checks cooldowns, and iterates through
// the command steps defined in the CampaignManifest. Each step's CEL formula is evaluated,
// and results are mapped to engine events. It supports adjudication and result chaining.
func ExecuteGenericCommand(cmdName string, actorID string, targets []string, params map[string]any, originalCmd string, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
	if params == nil {
		params = make(map[string]any)
	}

	var events []engine.Event
	// Hook into dice rolls to capture them as events
	reg.SetDiceReporter(func(dice string, result int) {
		events = append(events, &engine.DiceRolledEvent{
			ActorName: actorID,
			Total:     result,
			RawRolls:  []int{result},
		})
	})
	defer reg.SetDiceReporter(nil)
	cmdDef, ok := reg.GetCommand(cmdName)
	if !ok {
		fmt.Printf(">>> ERROR: cmd %s not found\n", cmdName)
		return nil, fmt.Errorf("command '%s' not found in manifest", cmdName)
	}

	actor, ok := state.Entities[actorID]
	if !ok {
		fmt.Printf(">>> ERROR: actor %s not found\n", actorID)
		return nil, fmt.Errorf("actor '%s' not found in encounter", actorID)
	}

	// Normalize standard parameters
	if _, ok := params["opportunity"]; !ok {
		params["opportunity"] = false
	}
	if _, ok := params["offhand"]; !ok {
		params["offhand"] = false
	}

	// Resolve actor-specific action data (e.g., weapon stats)
	weaponName, _ := params["weapon"].(string)
	if actionData, found := ResolveEntityAction(actorID, weaponName, loader); found {
		params["weapon_resolved"] = actionData.Name
		params["bonus"] = actionData.AttackBonus
		params["recharge"] = actionData.Recharge
		if actionData.HitRule != "" {
			params["hit_rule"] = actionData.HitRule
		}
		if _, ok := params["dice"]; !ok {
			params["dice"] = actionData.DamageDice
		}
		if _, ok := params["type"]; !ok {
			params["type"] = actionData.DamageType
		}
	} else {
		params["weapon_resolved"] = weaponName
		params["bonus"] = 0
		params["recharge"] = ""
		if _, ok := params["dice"]; !ok {
			params["dice"] = "1d4"
		}
		if _, ok := params["type"]; !ok {
			params["type"] = "bludgeoning"
		}
	}

	// Two-Weapon Fighting: don't add positive ability modifier to off-hand attack damage
	if oh, ok := params["offhand"].(bool); ok && oh {
		if dice, ok := params["dice"].(string); ok && strings.Contains(dice, "+") {
			params["dice"] = strings.TrimSpace(strings.Split(dice, "+")[0])
		}
	}

	// Check cooldown
	if resolvedName, ok := params["weapon_resolved"].(string); ok && resolvedName != "" {
		if recharge, ok := params["recharge"].(string); ok && recharge != "" {
			if spent, ok := state.SpentRecharges[actorID]; ok {
				for _, s := range spent {
					if s == resolvedName {
						return nil, fmt.Errorf("ability '%s' is cooling down", resolvedName)
					}
				}
			}
		}
	}

	// events is already initialized above

	for _, targetID := range targets {
		target, _ := state.Entities[targetID]
		evalCtx := rules.BuildEvalContext(state, actor, target, params)
		evalCtx["manifest"] = map[string]any{
			"approved": state.PendingAdjudication != nil && state.PendingAdjudication.Approved,
		}
		steps := map[string]any{}
		evalCtx["steps"] = steps

		// Map results from steps into events
		for _, step := range cmdDef.Steps {
			res, err := reg.Eval(step.Formula, evalCtx)
			if err != nil {
				return nil, fmt.Errorf("command '%s' step '%s' failed: %w", cmdName, step.Name, err)
			}

			// Special case: check for adjudication request
			if s, ok := res.(string); ok && s == "adjudicate" {
				return []engine.Event{
					&engine.AdjudicationStartedEvent{
						OriginalCommand: originalCmd,
					},
				}, nil
			}

			// Special case: check for error request
			if s, ok := res.(string); ok && strings.HasPrefix(s, "error:") {
				return nil, fmt.Errorf("%s", strings.TrimPrefix(s, "error:"))
			}

			// Produce event if specified
			if step.Event != "" {
				event := mapManifestEvent(step.Event, actorID, targetID, res, evalCtx, params, cmdName, state)
				if event != nil {
					events = append(events, event)
				}
			}

			// If a boolean step returns false, we stop execution of this branch.
			// This ONLY applies for steps without an event (requirement checks).
			// Steps with events (like 'hit') allow subsequent steps to check their result.
			if b, ok := res.(bool); ok && !b && step.Event == "" {
				break
			}

			steps[step.Name] = res
		}
	}

	// Post-processing for specific commands
	if cmdName == "turn" {
		// If we just changed the turn, handle recharges for the NEW actor
		var turnChanged *engine.TurnChangedEvent
		for _, e := range events {
			if tc, ok := e.(*engine.TurnChangedEvent); ok {
				turnChanged = tc
				break
			}
		}

		if turnChanged != nil {
			nextActor := turnChanged.ActorID
			if spent, ok := state.SpentRecharges[nextActor]; ok && len(spent) > 0 {
				mon, err := loader.LoadMonster(nextActor)
				if err == nil {
					for _, actionName := range spent {
						for _, a := range mon.Actions {
							if strings.EqualFold(a.Name, actionName) && a.Recharge != "" {
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
		}
	}

	return events, nil
}

// mapManifestEvent converts a string event name and CEL result into a concrete engine.Event.
// It extracts relevant metadata from the evaluation context and parameters to populate
// event fields like targets, hit status, and resource usage.
func mapManifestEvent(eventName string, actorID, targetID string, res any, ctx map[string]any, params map[string]any, cmdName string, state *engine.GameState) engine.Event {
	switch eventName {
	case "AttackResolved":
		hit := false
		if b, ok := res.(bool); ok {
			hit = b
		}
		weapon := "unknown"
		if w, ok := params["weapon_resolved"]; ok {
			weapon = fmt.Sprintf("%v", w)
		} else if w, ok := params["weapon"]; ok {
			weapon = fmt.Sprintf("%v", w)
		}

		offhand := false
		if val, ok := params["offhand"]; ok {
			if b, ok := val.(bool); ok {
				offhand = b
			}
		}

		opportunity := false
		if val, ok := params["opportunity"]; ok {
			if b, ok := val.(bool); ok {
				opportunity = b
			}
		}

		return &engine.AttackResolvedEvent{
			Attacker:      actorID,
			Weapon:        weapon,
			Targets:       []string{targetID},
			HitStatus:     map[string]bool{targetID: hit},
			IsOffHand:     offhand,
			IsOpportunity: opportunity,
		}
	case "CheckResolved":
		success := true
		if b, ok := res.(bool); ok {
			success = b
		}
		val := 0
		if i, ok := res.(int64); ok {
			val = int(i)
		}

		// Fallback: search context for score if not in res
		if val == 0 {
			if steps, ok := ctx["steps"].(map[string]any); ok {
				if score, ok := steps["total"].(int64); ok {
					val = int(score)
				} else if score, ok := steps["score"].(int64); ok {
					val = int(score)
				}
			}
		}

		return &engine.CheckResolvedEvent{
			ActorID: actorID,
			Result:  val,
			Success: success,
		}
	case "AdjudicationStarted":
		return &engine.AdjudicationStartedEvent{
			OriginalCommand: fmt.Sprintf("%s attempted to %s on %s", actorID, cmdName, targetID),
		}
	case "GrappleTaken":
		return &engine.GrappleTakenEvent{
			Attacker: actorID,
			Target:   targetID,
		}
	case "AskIssued":
		dc := 10
		if d, ok := params["dc"]; ok {
			if i, ok := d.(int); ok {
				dc = i
			} else if i, ok := d.(int64); ok {
				dc = int(i)
			}
		} else if steps, ok := ctx["steps"].(map[string]any); ok {
			if d, ok := steps["dc"]; ok {
				if i, ok := d.(int); ok {
					dc = i
				} else if i, ok := d.(int64); ok {
					dc = int(i)
				}
			}
		}
		check := []string{"strength", "save", "or", "dexterity", "save"}
		if c, ok := params["check"]; ok {
			if s, ok := c.([]string); ok {
				check = s
			} else if s, ok := c.(string); ok {
				check = strings.Split(s, " ")
			}
		}

		fails := &engine.RollConsequence{}
		if cmdName == "grapple" {
			fails.Condition = "grappledby:" + actorID
		}

		return &engine.AskIssuedEvent{
			Targets: []string{targetID},
			Check:   check,
			DC:      dc,
			Fails:   fails,
		}
	case "Hint":
		return &engine.HintEvent{
			MessageStr: fmt.Sprintf("%s attempted to %s %s", actorID, cmdName, targetID),
		}
	case "ActionConsumed":
		return &engine.ActionConsumedEvent{
			ActorID: actorID,
		}
	case "HPChanged":
		amount := 0
		if i, ok := res.(int64); ok {
			amount = int(i)
		}
		return &engine.HPChangedEvent{
			ActorID: targetID,
			Amount:  amount,
		}
	case "AbilitySpent":
		if b, ok := res.(bool); ok && !b {
			return nil
		}
		weapon := "unknown"
		if w, ok := params["weapon_resolved"]; ok {
			weapon = fmt.Sprintf("%v", w)
		} else if w, ok := params["weapon"]; ok {
			weapon = fmt.Sprintf("%v", w)
		}
		return &engine.AbilitySpentEvent{
			ActorID:    actorID,
			ActionName: weapon,
		}
	case "ConditionRemoved":
		if s, ok := res.(string); ok && s != "" && s != "none" && s != "ok" {
			return &engine.ConditionRemovedEvent{
				ActorID:   actorID,
				Condition: s,
			}
		}
	case "TurnEnded":
		return &engine.TurnEndedEvent{
			ActorID: actorID,
		}
	case "TurnChanged":
		nextActor := actorID
		if s, ok := res.(string); ok && s != "" && s != "ok" && s != "true" {
			nextActor = s
		} else if state != nil && len(state.TurnOrder) > 0 {
			// Auto-calculate next actor if not specified by manifest
			idx := -1
			for i, id := range state.TurnOrder {
				if id == actorID {
					idx = i
					break
				}
			}
			if idx == -1 {
				idx = state.CurrentTurn
			}
			nextIdx := (idx + 1) % len(state.TurnOrder)
			nextActor = state.TurnOrder[nextIdx]
		}
		return &engine.TurnChangedEvent{
			ActorID: nextActor,
		}
	case "DodgeTaken":
		return &engine.DodgeTakenEvent{
			ActorID: actorID,
		}
	case "InitiativeRolled":
		score := 0
		if i, ok := res.(int64); ok {
			score = int(i)
		}
		if score == 0 {
			if steps, ok := ctx["steps"].(map[string]any); ok {
				if v, ok := steps["roll"].(int64); ok {
					score = int(v)
				}
			}
		}
		return &engine.InitiativeRolledEvent{
			ActorName: actorID,
			Score:     score,
		}
	case "RechargeRolled":
		if m, ok := res.(map[string]any); ok {
			return &engine.RechargeRolledEvent{
				ActorID:     actorID,
				ActionName:  fmt.Sprintf("%v", m["action"]),
				Roll:        int(m["roll"].(int64)),
				Requirement: fmt.Sprintf("%v", m["requirement"]),
				Success:     m["success"].(bool),
			}
		}
	case "AbilityRecharged":
		if s, ok := res.(string); ok && s != "" && s != "ok" {
			return &engine.AbilityRechargedEvent{
				ActorID:    actorID,
				ActionName: s,
			}
		}
	}
	return nil
}

// RollInitiative is a legacy-compatible helper that uses the current rules to roll initiative for an actor.
// It's used by ExecuteAdd and ExecuteEncounter.
func RollInitiative(id string, stats TargetRes, reg *rules.Registry) engine.Event {
	// Baseline as we don't always have the rollFunc here easily without reg
	dex := 10
	if stats.Name != "" {
		if d, ok := stats.Stats["dex"]; ok {
			dex = d
		}
	}
	mod := (dex - 10) / 2
	roll := 10 // Baseline as we don't always have the rollFunc here easily without reg
	if reg != nil {
		// Mock context
		ctx := map[string]any{
			"actor": map[string]any{
				"stats": map[string]any{
					"dex": dex,
				},
			},
		}
		if res, err := reg.Eval("roll('1d20') + mod(actor.stats.dex)", ctx); err == nil {
			if i, ok := res.(int64); ok {
				return &engine.InitiativeRolledEvent{
					ActorName: id,
					Score:     int(i),
				}
			}
		}
	}
	return &engine.InitiativeRolledEvent{
		ActorName: id,
		Score:     roll + mod,
	}
}
