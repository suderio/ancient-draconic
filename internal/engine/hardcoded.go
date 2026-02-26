package engine

import (
	"fmt"
	"strings"
)

// hardcodedCommands lists the commands that exist in every game regardless of manifest.
var hardcodedCommands = map[string]bool{
	"roll":       true,
	"help":       true,
	"hint":       true,
	"ask":        true,
	"adjudicate": true,
	"allow":      true,
	"deny":       true,
}

// isHardcoded returns true if the command is a built-in that is not defined in the manifest.
func isHardcoded(cmdName string) bool {
	return hardcodedCommands[cmdName]
}

// executeHardcoded dispatches a built-in command.
func executeHardcoded(
	cmdName string,
	actorID string,
	targets []string,
	params map[string]any,
	state *GameState,
	m *Manifest,
	eval *Evaluator,
) ([]Event, error) {
	switch cmdName {
	case "roll":
		return executeRoll(actorID, params, eval)
	case "help":
		return executeHelp(params, m)
	case "hint":
		return executeHint(state, m)
	case "ask":
		return executeAsk(actorID, targets, params)
	case "allow":
		return executeAllow(actorID, state)
	case "deny":
		return executeDeny(actorID, state)
	case "adjudicate":
		return executeAdjudicate(actorID, state)
	}
	return nil, fmt.Errorf("unknown hardcoded command: %s", cmdName)
}

// executeRoll evaluates a dice expression and returns a DiceRolledEvent.
// Expected params: {"dice": "2d6+3"}
func executeRoll(actorID string, params map[string]any, eval *Evaluator) ([]Event, error) {
	dice, ok := params["dice"].(string)
	if !ok || dice == "" {
		return nil, fmt.Errorf("roll requires a 'dice' parameter (e.g., roll dice: 2d6+3)")
	}
	result := eval.rollFunc(dice)
	return []Event{
		&DiceRolledEvent{ActorID: actorID, Dice: dice, Result: result},
	}, nil
}

// executeHelp returns help text from the manifest.
// If params contains "command", show help for that specific command.
// Otherwise, list all available commands.
func executeHelp(params map[string]any, m *Manifest) ([]Event, error) {
	if cmdName, ok := params["command"].(string); ok && cmdName != "" {
		// Help for a specific command
		if cmd, ok := m.Commands[cmdName]; ok {
			helpText := fmt.Sprintf("**%s**: %s\nUsage: %s", cmd.Name, cmd.Help, cmd.Error)
			return []Event{&HintEvent{MessageStr: helpText}}, nil
		}
		// Try underscore variant (e.g., "encounter start" → "encounter_start")
		underscore := strings.ReplaceAll(cmdName, " ", "_")
		if cmd, ok := m.Commands[underscore]; ok {
			helpText := fmt.Sprintf("**%s**: %s\nUsage: %s", cmd.Name, cmd.Help, cmd.Error)
			return []Event{&HintEvent{MessageStr: helpText}}, nil
		}
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}

	// List all commands
	var lines []string
	lines = append(lines, "**Available commands:**")
	// Hardcoded commands
	lines = append(lines, "  roll, help, hint, ask, adjudicate, allow, deny")
	// Manifest commands
	for _, cmd := range m.Commands {
		lines = append(lines, fmt.Sprintf("  **%s** — %s", cmd.Name, cmd.Help))
	}
	return []Event{&HintEvent{MessageStr: strings.Join(lines, "\n")}}, nil
}

// executeHint returns the hint text from the last executed command.
func executeHint(state *GameState, m *Manifest) ([]Event, error) {
	if state.LastCommand == "" {
		return []Event{&HintEvent{MessageStr: "No command has been executed yet."}}, nil
	}
	if cmd, ok := m.Commands[state.LastCommand]; ok && cmd.Hint != "" {
		return []Event{&HintEvent{MessageStr: cmd.Hint}}, nil
	}
	return []Event{&HintEvent{MessageStr: fmt.Sprintf("No hint available for '%s'.", state.LastCommand)}}, nil
}

// executeAsk emits an AskIssuedEvent to request input from targets.
// Expected params: {"options": ["command1", "command2"]}
func executeAsk(actorID string, targets []string, params map[string]any) ([]Event, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("ask requires at least one target")
	}

	options, _ := params["options"].([]string)
	if len(options) == 0 {
		// Try []any
		if anyOpts, ok := params["options"].([]any); ok {
			for _, o := range anyOpts {
				if s, ok := o.(string); ok {
					options = append(options, s)
				}
			}
		}
	}

	var events []Event
	for _, target := range targets {
		events = append(events, &AskIssuedEvent{TargetID: target, Options: options})
	}
	return events, nil
}

// executeAllow resolves a pending ask by approving it.
func executeAllow(actorID string, state *GameState) ([]Event, error) {
	if !isGM(actorID) {
		return nil, fmt.Errorf("only the GM can use 'allow'")
	}
	// Clear pending ask and record approval
	delete(state.Metadata, "pending_ask")
	return []Event{
		&MetadataChangedEvent{Key: "last_adjudication", Value: map[string]any{"approved": true}},
	}, nil
}

// executeDeny resolves a pending ask by rejecting it.
func executeDeny(actorID string, state *GameState) ([]Event, error) {
	if !isGM(actorID) {
		return nil, fmt.Errorf("only the GM can use 'deny'")
	}
	delete(state.Metadata, "pending_ask")
	return []Event{
		&MetadataChangedEvent{Key: "last_adjudication", Value: map[string]any{"approved": false}},
	}, nil
}

// executeAdjudicate is similar to allow but may involve more context.
// For now, it behaves like allow.
func executeAdjudicate(actorID string, state *GameState) ([]Event, error) {
	return executeAllow(actorID, state)
}
