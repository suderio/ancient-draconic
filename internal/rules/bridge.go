package rules

import (
	"github.com/suderio/ancient-draconic/internal/engine"
)

// ContextFromEntity converts an engine.Entity into a map suitable for CEL evaluation.
func ContextFromEntity(e *engine.Entity) map[string]any {
	if e == nil {
		return nil
	}
	return map[string]any{
		"id":            e.ID,
		"name":          e.Name,
		"types":         e.Types,
		"classes":       e.Classes,
		"stats":         e.Stats,
		"resources":     e.Resources,
		"spent":         e.Spent,
		"conditions":    e.Conditions,
		"proficiencies": e.Proficiencies,
		"statuses":      e.Statuses,
		"inventory": func() map[string]int {
			if e.Inventory != nil {
				return e.Inventory
			}
			return make(map[string]int)
		}(),
		"profs":           e.Proficiencies,
		"size":            e.Classes["size"],
		"category":        e.Classes["category"],
		"immunities":      e.Immunities,
		"resistances":     e.Resistances,
		"vulnerabilities": e.Vulnerabilities,
		"actions_remaining": func() int {
			if e.Spent != nil && e.Resources != nil {
				return e.Resources["actions"] - e.Spent["actions"]
			}
			return 0
		}(),
	}
}

// BuildEvalContext creates a standard RPG context with actor, target, action, and system and state metadata.
func BuildEvalContext(state *engine.GameState, actor *engine.Entity, target *engine.Entity, action map[string]any) map[string]any {
	res := map[string]any{
		"actor":  ContextFromEntity(actor),
		"target": ContextFromEntity(target),
		"action": action,
	}

	if state != nil {
		// Ensure critical keys always exist to avoid CEL errors
		res["pending_adjudication"] = map[string]any{"approved": false}
		res["pending_checks"] = map[string]any{}
		res["pending_damage"] = map[string]any{}

		// Expose all metadata keys directly in the context (overwrites defaults if present)
		for k, v := range state.Metadata {
			res[k] = v
		}

		res["is_frozen"] = state.IsFrozen()
		res["is_encounter_active"] = state.IsEncounterActive
		res["entities"] = state.Entities
	}

	return res
}
