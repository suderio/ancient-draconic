package command

import (
	"fmt"
	"strings"

	"dndsl/internal/engine"
	"dndsl/internal/parser"
)

type commandHelp struct {
	Usage   string
	Summary string
	GMOnly  bool
}

var commandRegistry = map[string]commandHelp{
	"roll": {
		Usage:   "roll [:by Actor] <dice>",
		Summary: "Calculates dice expressions (e.g., 3d6+2).",
	},
	"encounter": {
		Usage:   "encounter [:by GM] <start|end> [:with T1 [:and T2]*]",
		Summary: "Starts or ends a combat encounter. GM only.",
		GMOnly:  true,
	},
	"add": {
		Usage:   "add [:by GM] T1 [:and T2]*",
		Summary: "Adds participants to an active encounter. GM only.",
		GMOnly:  true,
	},
	"initiative": {
		Usage:   "initiative [:by Actor] [manual_score]",
		Summary: "Sets or rolls initiative for a participant.",
	},
	"attack": {
		Usage:   "attack [:by Actor] :with Weapon :to Target1 [:and Target2]* [:dice 1d20+M]",
		Summary: "Attempts to strike targets with a weapon.",
	},
	"damage": {
		Usage:   "damage [:by Actor] [:with Weapon] [:dice Dice :type Type]*",
		Summary: "Resolves HP reduction after a successful strike.",
	},
	"turn": {
		Usage:   "turn [:by Actor]",
		Summary: "Ends the current actor's turn and rotates to the next.",
	},
	"ask": {
		Usage:   "ask :by GM :check skill :of participants :dc N [:fails cons] [:succeeds cons]",
		Summary: "GM requests a check from participants. GM only.",
		GMOnly:  true,
	},
	"check": {
		Usage:   "check :by Actor <skill/save>",
		Summary: "Resolves an ability check or saving throw.",
	},
	"hint": {
		Usage:   "hint",
		Summary: "Provides mechanical guidance based on the current context.",
	},
	"help": {
		Usage:   "help [:by Actor] [command|all]",
		Summary: "Shows available commands or detailed info on a specific one.",
	},
}

// ExecuteHelp provides contextual guidance on DSL command usage
func ExecuteHelp(cmd *parser.HelpCmd, state *engine.GameState) ([]engine.Event, error) {
	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}
	isGM := strings.EqualFold(actorName, "GM")

	// 1. Detailed help for a specific command
	if cmd.Command != "" && !strings.EqualFold(cmd.Command, "all") {
		help, ok := commandRegistry[strings.ToLower(cmd.Command)]
		if !ok {
			return nil, fmt.Errorf("Unknown command: %s", cmd.Command)
		}
		if help.GMOnly && !isGM {
			return nil, fmt.Errorf("The command %s is only available to the GM", cmd.Command)
		}

		msg := fmt.Sprintf("Command: %s\nUsage: %s\nSummary: %s", strings.ToLower(cmd.Command), help.Usage, help.Summary)
		return []engine.Event{&engine.HintEvent{MessageStr: msg}}, nil
	}

	// 2. List all (or all player) commands
	if strings.EqualFold(cmd.Command, "all") {
		var sb strings.Builder
		sb.WriteString("Available Commands:\n")

		// Sort keys for deterministic output
		keys := []string{"roll", "encounter", "add", "initiative", "attack", "damage", "turn", "ask", "check", "hint", "help"}
		for _, k := range keys {
			h := commandRegistry[k]
			if h.GMOnly && !isGM {
				continue
			}
			sb.WriteString(fmt.Sprintf(" - %s: %s\n", k, h.Summary))
		}
		return []engine.Event{&engine.HintEvent{MessageStr: strings.TrimSpace(sb.String())}}, nil
	}

	// 3. Context-Aware Help
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Help for %s in current context:\n", actorName))

	// Universal commands
	sb.WriteString(" - roll: Roll dice\n")
	sb.WriteString(" - hint: Get current context hint\n")
	sb.WriteString(" - help: This help system\n")

	if !state.IsEncounterActive {
		if isGM {
			sb.WriteString(" - encounter start: Start a new combat encounter\n")
		} else {
			sb.WriteString("Waiting for GM to start an encounter.\n")
		}
	} else {
		// Mid-encounter context
		if state.IsFrozen() {
			// Check why frozen
			if len(state.PendingChecks) > 0 {
				if _, ok := state.PendingChecks[actorName]; ok || (isGM && actorName == "GM") {
					sb.WriteString(" - check: Resolve your pending check\n")
				} else {
					sb.WriteString("Waiting for other participants to resolve checks.\n")
				}
			} else {
				// Frozen by initiative?
				sb.WriteString(" - initiative: Roll for combat order\n")
			}

			if isGM {
				sb.WriteString(" - encounter end: Stop combat\n")
				sb.WriteString(" - add: Add more actors to combat\n")
			}
		} else {
			// Turn-based combat
			currentActor := state.TurnOrder[state.CurrentTurn]
			canAct := isGM || strings.EqualFold(actorName, currentActor) || strings.EqualFold(actorName, strings.ReplaceAll(currentActor, "-", "_"))

			if canAct {
				sb.WriteString(" - attack: Attempt to hit someone\n")
				if state.PendingDamage != nil && strings.EqualFold(state.PendingDamage.Attacker, currentActor) {
					sb.WriteString(" - damage: Resolve weapon damage\n")
				}
				sb.WriteString(" - turn: End your turn\n")
			} else {
				sb.WriteString(fmt.Sprintf("Waiting for %s to act.\n", currentActor))
			}

			if isGM {
				sb.WriteString(" - ask: Request checks from players\n")
				sb.WriteString(" - encounter end: Stop combat\n")
			}
		}
	}

	sb.WriteString("\nUse 'help :by actor all' for a full list of commands.")
	return []engine.Event{&engine.HintEvent{MessageStr: strings.TrimSpace(sb.String())}}, nil
}
