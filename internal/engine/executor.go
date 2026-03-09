package engine

import (
	"fmt"
	"sort"
	"strings"
)

// ExecuteCommand is the main entry point for running a manifest-driven command.
// It follows the pipeline: restrictions → params → prereq → game → targets → actor.
func ExecuteCommand(
	cmdName string,
	actorID string,
	targets []string,
	params map[string]any,
	state *GameState,
	m *Manifest,
	eval *LuaEvaluator,
) ([]Event, error) {
	if isBuiltin(cmdName) {
		return executeBuiltin(cmdName, actorID, targets, params, state, m, eval)
	}

	cmdDef, ok := m.Commands[cmdName]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}

	if err := checkRestrictions(cmdName, actorID, m); err != nil {
		return nil, err
	}

	if err := validateParams(cmdDef, params); err != nil {
		return nil, fmt.Errorf("invalid parameters for %s: %w. Usage: %s", cmdDef.Name, err, cmdDef.Error)
	}

	actor := state.Entities[actorID]

	// Evaluate prereqs
	ctx := BuildContext(state, actor, nil, params, nil, nil, nil)
	for _, prereq := range cmdDef.Prereq {
		result, err := eval.Eval(prereq.Value, ctx)
		if err != nil {
			return nil, fmt.Errorf("prereq '%s' evaluation failed: %w", prereq.Name, err)
		}
		passed, ok := result.(bool)
		if !ok || !passed {
			return nil, fmt.Errorf("%s", prereq.Error)
		}
	}

	// Execute game steps (run once)
	var events []Event
	gameResults := make(map[string]any)
	for _, step := range cmdDef.Game {
		ctx = BuildContext(state, actor, nil, params, gameResults, nil, nil)
		result, err := eval.Eval(step.Value, ctx)
		if err != nil {
			return nil, fmt.Errorf("game step '%s' failed: %w", step.Name, err)
		}
		evts, plain := dispatchTaggedResult(result, actorID, "", cmdName, state)
		gameResults[step.Name] = plain
		events = append(events, evts...)
	}

	// Execute target steps (run per-target)
	allTargets := resolveTargets(cmdDef, targets, params)
	for _, targetID := range allTargets {
		target := state.Entities[targetID]
		targetResults := make(map[string]any)

		for _, step := range cmdDef.Targets {
			ctx = BuildContext(state, actor, target, params, gameResults, targetResults, nil)
			result, err := eval.Eval(step.Value, ctx)
			if err != nil {
				return nil, fmt.Errorf("target step '%s' for %s failed: %w", step.Name, targetID, err)
			}
			evts, plain := dispatchTaggedResult(result, actorID, targetID, cmdName, state)
			targetResults[step.Name] = plain
			events = append(events, evts...)
		}
	}

	// Execute actor steps (run once, affecting the actor)
	actorResults := make(map[string]any)
	for _, step := range cmdDef.Actor {
		ctx = BuildContext(state, actor, nil, params, gameResults, nil, actorResults)
		result, err := eval.Eval(step.Value, ctx)
		if err != nil {
			return nil, fmt.Errorf("actor step '%s' failed: %w", step.Name, err)
		}
		evts, plain := dispatchTaggedResult(result, actorID, "", cmdName, state)
		actorResults[step.Name] = plain
		events = append(events, evts...)
	}

	state.LastCommand = cmdName
	return events, nil
}

// dispatchTaggedResult inspects the Eval result. If it is a map with an `_event` key,
// it dispatches the appropriate Event(s) and returns them along with a clean value for step results.
// If there is no `_event` key, it returns (nil, result) — a pure computation step.
func dispatchTaggedResult(result any, actorID, targetID, cmdName string, state *GameState) ([]Event, any) {
	m, ok := result.(map[string]any)
	if !ok {
		return nil, result
	}
	eventType, ok := m["_event"].(string)
	if !ok {
		return nil, result
	}

	effectiveTarget := actorID
	if targetID != "" {
		effectiveTarget = targetID
	}

	switch eventType {
	case "loop":
		name, _ := m["name"].(string)
		if name == "" {
			name = cmdName
		}
		active, _ := m["active"].(bool)
		return []Event{&LoopEvent{LoopName: name, Active: active}}, active

	case "loop_order":
		name, _ := m["name"].(string)
		if name == "" {
			name = cmdName
		}
		ascending, _ := m["ascending"].(bool)
		return []Event{&LoopOrderAscendingEvent{LoopName: name, Ascending: ascending}}, ascending

	case "loop_value":
		name, _ := m["name"].(string)
		if name == "" {
			name = cmdName
		}
		value, ok := toInt(m["value"])
		if !ok {
			return nil, result
		}
		return []Event{&LoopOrderEvent{LoopName: name, ActorID: actorID, Value: value}}, value

	case "add_actor":
		actors := m["actors"]
		switch v := actors.(type) {
		case string:
			name, _ := m["name"].(string)
			if name == "" {
				name = cmdName
			}
			return []Event{&ActorAddedEvent{LoopName: name, ActorID: v}}, v
		case []any:
			if len(v) > 0 {
				if s, ok := v[0].(string); ok {
					name, _ := m["name"].(string)
					if name == "" {
						name = cmdName
					}
					return []Event{&ActorAddedEvent{LoopName: name, ActorID: s}}, actors
				}
			}
		}
		return nil, result

	case "ask":
		askTarget, _ := m["target"].(string)
		if askTarget == "" {
			askTarget = effectiveTarget
		}
		var options []string
		if opts, ok := m["options"].([]any); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok {
					options = append(options, s)
				}
			}
		}
		return []Event{&AskIssuedEvent{TargetID: askTarget, Options: options}}, m

	case "condition":
		cond, _ := m["condition"].(string)
		add, _ := m["add"].(bool)
		return []Event{&ConditionEvent{ActorID: effectiveTarget, Condition: cond, Add: add}}, cond

	case "spend":
		key, _ := m["key"].(string)
		amount, _ := toInt(m["amount"])
		return []Event{&AddSpentEvent{ActorID: actorID, Key: key, Amount: amount}}, key

	case "set_attr":
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
		return []Event{&AttributeChangedEvent{ActorID: target, Section: section, Key: key, Value: value}}, m

	case "contest":
		value, ok := toInt(m["value"])
		if !ok {
			return nil, result
		}
		return []Event{&MetadataChangedEvent{
			Key:   "contest",
			Value: map[string]any{"actor": actorID, "value": value},
		}}, map[string]any{"actor": actorID, "value": value}

	case "check":
		passed, _ := m["passed"].(bool)
		return []Event{&CheckEvent{ActorID: actorID, Check: cmdName, Passed: passed}}, passed

	case "hint":
		msg, _ := m["message"].(string)
		return []Event{&HintEvent{MessageStr: msg}}, msg

	case "metadata":
		key, _ := m["key"].(string)
		value := m["value"]
		return []Event{&MetadataChangedEvent{Key: key, Value: value}}, m

	case "next_turn":
		return dispatchNextTurn(m, actorID, cmdName, state)

	default:
		// Unknown event type — treat as CustomEvent
		// If emit() was used, the result has { _event, payload } — unwrap the payload.
		if p, ok := m["payload"].(map[string]any); ok {
			return []Event{&CustomEvent{EventType: eventType, ActorID: actorID, Payload: p}}, m
		}
		payload := make(map[string]any)
		for k, v := range m {
			if k != "_event" {
				payload[k] = v
			}
		}
		return []Event{&CustomEvent{EventType: eventType, ActorID: actorID, Payload: payload}}, m
	}
}

// dispatchNextTurn handles the next_turn tagged result by emitting multiple events:
// TurnEndedEvent → (RoundStartedEvent if wrap-around) → TurnStartedEvent.
func dispatchNextTurn(m map[string]any, actorID, cmdName string, state *GameState) ([]Event, any) {
	name, _ := m["name"].(string)
	if name == "" {
		name = cmdName
	}

	loop, ok := state.Loops[name]
	if !ok || !loop.Active {
		return nil, m
	}

	// Get sorted actor list
	sorted := sortedActors(loop)
	if len(sorted) == 0 {
		return nil, m
	}

	var events []Event

	// End current actor's turn
	currentActor := actorID
	if loop.Current >= 0 && loop.Current < len(sorted) {
		currentActor = sorted[loop.Current]
	}
	events = append(events, &TurnEndedEvent{LoopName: name, ActorID: currentActor})

	// Advance to next
	nextIdx := loop.Current + 1
	newRound := false
	if nextIdx >= len(sorted) {
		nextIdx = 0
		newRound = true
	}

	// Round wrap-around
	nextRound := loop.Round
	if newRound || loop.Round == 0 {
		nextRound = loop.Round + 1
		events = append(events, &RoundStartedEvent{LoopName: name, Round: nextRound})
	}

	// Start next turn
	nextActor := sorted[nextIdx]
	nextTurn := nextIdx + 1
	events = append(events, &TurnStartedEvent{LoopName: name, ActorID: nextActor, Turn: nextTurn})

	return events, map[string]any{"actor": nextActor, "turn": nextTurn, "round": nextRound}
}

// sortedActors returns the loop's actors sorted by their Order value.
func sortedActors(loop *Loop) []string {
	actors := make([]string, len(loop.Actors))
	copy(actors, loop.Actors)
	sort.Slice(actors, func(i, j int) bool {
		if loop.Ascending {
			return loop.Order[actors[i]] < loop.Order[actors[j]]
		}
		return loop.Order[actors[i]] > loop.Order[actors[j]]
	})
	return actors
}

func checkRestrictions(cmdName, actorID string, m *Manifest) error {
	for _, gmCmd := range m.Restrictions.GMCommands {
		if cmdName == gmCmd && !isGM(actorID) {
			return fmt.Errorf("unauthorized: %s can only be executed by the GM", cmdName)
		}
	}
	return nil
}

func isGM(actorID string) bool {
	return strings.ToUpper(actorID) == "GM"
}

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

func resolveTargets(cmd CommandDef, explicitTargets []string, params map[string]any) []string {
	seen := make(map[string]bool)
	var result []string

	for _, t := range explicitTargets {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}

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
