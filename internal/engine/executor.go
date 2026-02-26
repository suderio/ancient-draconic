package engine

import (
	"fmt"
	"strings"
)

// ExecuteCommand is the main entry point for running a manifest-driven command.
// It follows the pipeline: restrictions → params → prereq → game → targets → actor.
//
// Parameters:
//   - cmdName: the manifest key for the command (e.g., "encounter_start")
//   - actorID: the ID of the entity performing the command
//   - targets: list of target entity IDs
//   - params: parsed command parameters (e.g., {"skill": "athletics", "dc": 15})
//   - state: the current game state
//   - manifest: the loaded campaign manifest
//   - eval: the CEL evaluator
//
// Returns a list of events to apply and any error encountered.
func ExecuteCommand(
	cmdName string,
	actorID string,
	targets []string,
	params map[string]any,
	state *GameState,
	m *Manifest,
	eval *Evaluator,
) ([]Event, error) {
	// 1. Check if this is a hardcoded command
	if isHardcoded(cmdName) {
		return executeHardcoded(cmdName, actorID, targets, params, state, m, eval)
	}

	// 2. Look up the command definition
	cmdDef, ok := m.Commands[cmdName]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}

	// 3. Check restrictions
	if err := checkRestrictions(cmdName, actorID, m); err != nil {
		return nil, err
	}

	// 4. Validate parameters
	if err := validateParams(cmdDef, params); err != nil {
		return nil, fmt.Errorf("invalid parameters for %s: %w. Usage: %s", cmdDef.Name, err, cmdDef.Error)
	}

	// 5. Resolve actor entity
	actor := state.Entities[actorID]

	// 6. Build initial context and evaluate prereqs
	stepResults := make(map[string]any)
	ctx := BuildContext(state, actor, nil, params, stepResults, m)

	for _, prereq := range cmdDef.Prereq {
		result, err := eval.Eval(prereq.Formula, ctx)
		if err != nil {
			return nil, fmt.Errorf("prereq '%s' evaluation failed: %w", prereq.Name, err)
		}
		passed, ok := result.(bool)
		if !ok || !passed {
			return nil, fmt.Errorf("%s", prereq.Error)
		}
	}

	// 7. Execute game steps (run once)
	var events []Event
	gameResults := make(map[string]any)
	for _, step := range cmdDef.Game {
		ctx = BuildContext(state, actor, nil, params, gameResults, m)
		result, err := eval.Eval(step.Formula, ctx)
		if err != nil {
			return nil, fmt.Errorf("game step '%s' failed: %w", step.Name, err)
		}
		gameResults[step.Name] = result

		if step.Event != "" {
			evt := mapStepToEvent(step.Event, result, actorID, "", cmdName, step.Loop, state)
			if evt != nil {
				events = append(events, evt)
			}
		}
	}

	// 8. Execute target steps (run per-target)
	// Collect all target IDs from params of type "target" or "list<target>"
	allTargets := resolveTargets(cmdDef, targets, params)
	for _, targetID := range allTargets {
		target := state.Entities[targetID]
		targetStepResults := make(map[string]any)
		// Carry forward game results into target step context
		for k, v := range gameResults {
			targetStepResults[k] = v
		}

		for _, step := range cmdDef.Targets {
			ctx = BuildContext(state, actor, target, params, targetStepResults, m)
			result, err := eval.Eval(step.Formula, ctx)
			if err != nil {
				return nil, fmt.Errorf("target step '%s' for %s failed: %w", step.Name, targetID, err)
			}
			targetStepResults[step.Name] = result

			if step.Event != "" {
				evt := mapStepToEvent(step.Event, result, actorID, targetID, cmdName, step.Loop, state)
				if evt != nil {
					events = append(events, evt)
				}
			}
		}
	}

	// 9. Execute actor steps (run once, affecting the actor)
	actorStepResults := make(map[string]any)
	for k, v := range gameResults {
		actorStepResults[k] = v
	}
	for _, step := range cmdDef.Actor {
		ctx = BuildContext(state, actor, nil, params, actorStepResults, m)
		result, err := eval.Eval(step.Formula, ctx)
		if err != nil {
			return nil, fmt.Errorf("actor step '%s' failed: %w", step.Name, err)
		}
		actorStepResults[step.Name] = result

		if step.Event != "" {
			evt := mapStepToEvent(step.Event, result, actorID, "", cmdName, step.Loop, state)
			if evt != nil {
				events = append(events, evt)
			}
		}
	}

	// 10. Track last command for hint
	state.LastCommand = cmdName

	return events, nil
}

// checkRestrictions enforces cross-cutting rules: GM-only commands and adjudication.
func checkRestrictions(cmdName, actorID string, m *Manifest) error {
	// GM-only commands
	for _, gmCmd := range m.Restrictions.GMCommands {
		if cmdName == gmCmd && !isGM(actorID) {
			return fmt.Errorf("unauthorized: %s can only be executed by the GM", cmdName)
		}
	}
	return nil
}

// isGM checks if the actor is the Game Master.
func isGM(actorID string) bool {
	return strings.ToUpper(actorID) == "GM"
}

// validateParams checks that all required parameters are present and have valid types.
func validateParams(cmd CommandDef, params map[string]any) error {
	for _, p := range cmd.Params {
		if p.Required {
			if _, ok := params[p.Name]; !ok {
				return fmt.Errorf("missing required parameter: %s", p.Name)
			}
		}
	}
	return nil
}

// resolveTargets collects all target IDs from the command's explicit targets list
// and any params declared as "target" or "list<target>".
func resolveTargets(cmd CommandDef, explicitTargets []string, params map[string]any) []string {
	seen := make(map[string]bool)
	var result []string

	// Add explicit targets first
	for _, t := range explicitTargets {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}

	// Add targets from params
	for _, p := range cmd.Params {
		if p.Type == "target" || p.Type == "list<target>" {
			if val, ok := params[p.Name]; ok {
				switch v := val.(type) {
				case string:
					if !seen[v] {
						seen[v] = true
						result = append(result, v)
					}
				case []string:
					for _, t := range v {
						if !seen[t] {
							seen[t] = true
							result = append(result, t)
						}
					}
				case []any:
					for _, t := range v {
						if s, ok := t.(string); ok && !seen[s] {
							seen[s] = true
							result = append(result, s)
						}
					}
				}
			}
		}
	}

	return result
}

// mapStepToEvent converts a step's event name and CEL result into a concrete Event.
// This is the central mapping from manifest event names to engine Event structs.
func mapStepToEvent(eventName string, result any, actorID, targetID, cmdName, loopOverride string, state *GameState) Event {
	// Derive the loop name: use explicit override if set, otherwise the command name.
	loopName := cmdName
	if loopOverride != "" {
		loopName = loopOverride
	}

	switch eventName {
	case "LoopEvent":
		active, _ := result.(bool)
		return &LoopEvent{LoopName: loopName, Active: active}

	case "LoopOrderAscendingEvent":
		ascending, _ := result.(bool)
		return &LoopOrderAscendingEvent{LoopName: loopName, Ascending: ascending}

	case "LoopOrderEvent":
		value, ok := toInt(result)
		if !ok {
			return nil
		}
		return &LoopOrderEvent{LoopName: loopName, ActorID: actorID, Value: value}

	case "ActorAddedEvent":
		// Result can be a single target ID or a list of IDs
		switch v := result.(type) {
		case string:
			return &ActorAddedEvent{LoopName: loopName, ActorID: v}
		case []any:
			// For lists, we return only the first event here.
			// The caller should handle multi-actor addition differently if needed.
			// TODO: Consider returning multiple events for list results.
			if len(v) > 0 {
				if s, ok := v[0].(string); ok {
					return &ActorAddedEvent{LoopName: loopName, ActorID: s}
				}
			}
		case []string:
			if len(v) > 0 {
				return &ActorAddedEvent{LoopName: loopName, ActorID: v[0]}
			}
		}
		return nil

	case "AttributeChangedEvent":
		m, ok := result.(map[string]any)
		if !ok {
			return nil
		}
		section, _ := m["section"].(string)
		key, _ := m["key"].(string)
		value := m["value"]
		target := actorID
		if targetID != "" {
			target = targetID
		}
		if aid, ok := m["actor_id"].(string); ok {
			target = aid
		}
		return &AttributeChangedEvent{ActorID: target, Section: section, Key: key, Value: value}

	case "AddSpentEvent":
		// The formula result is the key name to increment (e.g., "actions")
		key, ok := result.(string)
		if !ok {
			return nil
		}
		return &AddSpentEvent{ActorID: actorID, Key: key}

	case "AddConditionEvent":
		condition, ok := result.(string)
		if !ok {
			return nil
		}
		target := actorID
		if targetID != "" {
			target = targetID
		}
		return &ConditionEvent{ActorID: target, Condition: condition, Add: true}

	case "RemoveConditionEvent":
		condition, ok := result.(string)
		if !ok {
			return nil
		}
		target := actorID
		if targetID != "" {
			target = targetID
		}
		return &ConditionEvent{ActorID: target, Condition: condition, Add: false}

	case "AskIssuedEvent":
		// Result should be a list: [targetID, option1, option2, ...]
		list, ok := result.([]any)
		if !ok || len(list) < 2 {
			return nil
		}
		askTarget, _ := list[0].(string)
		if askTarget == "" {
			// If the first element is an entity map, extract the id
			if m, ok := list[0].(map[string]any); ok {
				askTarget, _ = m["id"].(string)
			}
		}
		options := make([]string, 0, len(list)-1)
		for _, opt := range list[1:] {
			if s, ok := opt.(string); ok {
				options = append(options, s)
			}
		}
		return &AskIssuedEvent{TargetID: askTarget, Options: options}

	case "HintEvent":
		msg, ok := result.(string)
		if !ok {
			return nil
		}
		return &HintEvent{MessageStr: msg}

	case "MetadataChangedEvent":
		m, ok := result.(map[string]any)
		if !ok {
			return nil
		}
		key, _ := m["key"].(string)
		value := m["value"]
		return &MetadataChangedEvent{Key: key, Value: value}

	case "CheckEvent":
		passed, ok := result.(bool)
		if !ok {
			return nil
		}
		return &CheckEvent{ActorID: actorID, Check: cmdName, Passed: passed}

	case "ContestStarted":
		// For now, treat as a metadata change recording the contest result
		value, ok := toInt(result)
		if !ok {
			return nil
		}
		return &MetadataChangedEvent{
			Key:   "contest",
			Value: map[string]any{"actor": actorID, "value": value},
		}

	case "ContestResolvedEvent":
		// Result is a boolean indicating if the contest was won
		passed, ok := result.(bool)
		if !ok {
			return nil
		}
		return &CheckEvent{ActorID: actorID, Check: "contest", Passed: passed}
	}

	return nil
}
