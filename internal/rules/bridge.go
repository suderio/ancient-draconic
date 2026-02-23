package rules

import (
	"github.com/suderio/ancient-draconic/internal/engine"
)

// ContextFromEntity converts an engine.Entity into a map suitable for CEL evaluation.
func ContextFromEntity(e *engine.Entity) map[string]any {
	if e == nil {
		return nil
	}
	profs := e.Proficiencies
	if profs == nil {
		profs = []string{}
	}
	// Provide default stats to avoid CEL evaluation errors
	stats := map[string]int{
		"str": 10, "dex": 10, "con": 10, "int": 10, "wis": 10, "cha": 10, "ac": 10, "prof_bonus": 0,
	}
	for k, v := range e.Stats {
		stats[k] = v
	}

	resistances := []any{}
	immunities := []any{}
	vulnerabilities := []any{}
	for _, d := range e.Defenses {
		for _, r := range d.Resistances {
			resistances = append(resistances, r)
		}
		for _, i := range d.Immunities {
			immunities = append(immunities, i)
		}
		for _, v := range d.Vulnerabilities {
			vulnerabilities = append(vulnerabilities, v)
		}
	}

	res := map[string]any{
		"id":                        e.ID,
		"name":                      e.Name,
		"hp":                        e.HP,
		"max_hp":                    e.MaxHP,
		"category":                  e.Category,
		"type":                      e.EntityType,
		"stats":                     stats,
		"resources":                 e.Resources,
		"conditions":                e.Conditions,
		"profs":                     profs,
		"actions_remaining":         e.ActionsRemaining,
		"bonus_actions_remaining":   e.BonusActionsRemaining,
		"reactions_remaining":       e.ReactionsRemaining,
		"attacks_remaining":         e.AttacksRemaining,
		"has_attacked_this_turn":    e.HasAttackedThisTurn,
		"last_attacked_with_weapon": e.LastAttackedWithWeapon,
		"resistances":               resistances,
		"immunities":                immunities,
		"vulnerabilities":           vulnerabilities,
		"size":                      e.Size,
	}
	return res
}

// BuildEvalContext creates a standard RPG context with actor, target, action, and system and state metadata.
func BuildEvalContext(state *engine.GameState, actor *engine.Entity, target *engine.Entity, action map[string]any) map[string]any {
	actorCtx := ContextFromEntity(actor)
	profs := []string{}
	if actorCtx != nil && actorCtx["profs"] != nil {
		profs = actorCtx["profs"].([]string)
	}

	res := map[string]any{
		"actor":  actorCtx,
		"target": ContextFromEntity(target),
		"action": action,
		"profs":  profs,
	}

	if state != nil {
		res["spent_recharges"] = state.SpentRecharges
		res["is_frozen"] = state.IsFrozen()
		if state.PendingAdjudication != nil {
			res["pending_adjudication"] = map[string]any{
				"original_command": state.PendingAdjudication.OriginalCommand,
				"approved":         state.PendingAdjudication.Approved,
			}
		} else {
			res["pending_adjudication"] = map[string]any{ // Use empty map instead of nil to avoid CEL issues
				"approved": false,
			}
		}
	}

	return res
}
