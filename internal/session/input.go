package session

import (
	"strings"
)

// ParsedInput represents the structured result of parsing a raw command string.
// The DSL format is:
//
//	<command> [by: <actor>] [<key>: <value> [and <value>]*]*
//
// Multi-word commands (e.g., "encounter start") are joined with underscores
// to match the manifest key format (e.g., "encounter_start").
type ParsedInput struct {
	Command string
	ActorID string
	Targets []string
	Params  map[string]any
}

// ParseInput parses a raw command string into a structured ParsedInput.
//
// Examples:
//
//	"roll dice: 2d6" → Command="roll", Params={"dice":"2d6"}
//	"attack by: Fighter to: Goblin" → Command="attack", ActorID="Fighter", Targets=["Goblin"]
//	"encounter start" → Command="encounter_start"
//	"grapple by: Fighter to: Goblin" → Command="grapple", ActorID="Fighter", Targets=["Goblin"]
//	"encounter start with: Fighter and Goblin" → Command="encounter_start", Params={"with":["Fighter","Goblin"]}
func ParseInput(input string) ParsedInput {
	result := ParsedInput{
		Params: make(map[string]any),
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return result
	}

	// Tokenize by splitting on spaces, then reconstruct key: value pairs.
	// We scan left-to-right, collecting tokens into the current key's values.
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return result
	}

	// Phase 1: Extract command words (everything before the first "key:" token)
	var cmdParts []string
	i := 0
	for i < len(tokens) {
		if strings.HasSuffix(tokens[i], ":") {
			break
		}
		cmdParts = append(cmdParts, tokens[i])
		i++
	}

	// The command is formed by joining the initial words with underscores
	result.Command = strings.ToLower(strings.Join(cmdParts, "_"))

	// Phase 2: Parse key: value pairs
	var currentKey string
	var currentValues []string

	flushKey := func() {
		if currentKey == "" {
			return
		}
		switch currentKey {
		case "by":
			if len(currentValues) > 0 {
				result.ActorID = currentValues[0]
			}
		case "to", "of":
			result.Targets = append(result.Targets, currentValues...)
		case "with":
			// "with" can serve as both targets and params, depending on context.
			// Store as param; the executor's resolveTargets will pick them up
			// if the command defines a "with" param as type "list<target>".
			if len(currentValues) == 1 {
				result.Params[currentKey] = currentValues[0]
			} else if len(currentValues) > 1 {
				result.Params[currentKey] = currentValues
			}
		default:
			if len(currentValues) == 1 {
				result.Params[currentKey] = currentValues[0]
			} else if len(currentValues) > 1 {
				result.Params[currentKey] = currentValues
			}
		}
		currentKey = ""
		currentValues = nil
	}

	for i < len(tokens) {
		token := tokens[i]
		if strings.HasSuffix(token, ":") {
			// Flush previous key
			flushKey()
			currentKey = strings.ToLower(strings.TrimSuffix(token, ":"))
		} else if strings.ToLower(token) == "and" {
			// "and" is a value separator, skip it
		} else {
			currentValues = append(currentValues, token)
		}
		i++
	}
	flushKey()

	// Default actor to GM if not specified
	if result.ActorID == "" {
		result.ActorID = "GM"
	}

	return result
}
