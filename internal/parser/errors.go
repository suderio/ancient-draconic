package parser

import (
	"fmt"
	"strings"
)

// MapError takes a raw input and a participle error, and returns a human-friendly guidance message.
func MapError(input string, err error) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("I wasn't able to understand your command")
	}

	parts := strings.Fields(strings.ToLower(input))
	cmd := parts[0]

	switch cmd {
	case "roll":
		return fmt.Errorf("The command roll must be: roll [:by Actor] <dice>")
	case "encounter":
		return fmt.Errorf("The command encounter must be: encounter [:by GM] <start|end> [:with Target1 [:and Target2]*]")
	case "add":
		return fmt.Errorf("The command add must be: add [:by GM] Target1 [:and Target2]*")
	case "initiative":
		return fmt.Errorf("The command initiative must be: initiative [:by Actor] [manual_score]")
	case "attack":
		return fmt.Errorf("The command attack must be: attack [:by Actor] :with Weapon :to Target1 [:and Target2]* [:dice 1d20+M]")
	case "damage":
		return fmt.Errorf("The command damage must be: damage [:by Actor] [:with Weapon] [:dice Dice :type Type]*")
	case "turn":
		return fmt.Errorf("The command turn must be: turn [:by Actor]")
	case "ask":
		return fmt.Errorf("The command ask must be: ask :by GM :check <skill> :of <targets> :dc <number> [:fails <consequence>] [:succeeds <consequence>]")
	case "check":
		return fmt.Errorf("The command check must be: check :by Actor <skill/save>")
	case "hint":
		return fmt.Errorf("The command hint must be: hint")
	}

	return fmt.Errorf("I wasn't able to understand your command")
}
