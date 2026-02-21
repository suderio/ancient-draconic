package command

import (
	"fmt"
	"strings"

	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/parser"
)

// Result encapsulation for tracking target types
type TargetRes struct {
	Type          string // 'Character' | 'Monster'
	Name          string
	InitiativeMod int
}

// ValidateGM simply asserts the executing actor matches "GM" (case-insensitive for leeway)
func ValidateGM(actor *parser.ActorExpr) error {
	if actor == nil || strings.ToUpper(actor.Name) != "GM" {
		return fmt.Errorf("unauthorized: this command can only be executed by the GM")
	}
	return nil
}

// CheckEntityLocally attempts to find the file resolution logic required in the specs
func CheckEntityLocally(target string, loader *data.Loader) (TargetRes, error) {
	if char, err := loader.LoadCharacter(target); err == nil {
		mod := data.CalculateModifier(char.Dexterity)
		return TargetRes{Type: "Character", Name: char.Name, InitiativeMod: mod}, nil
	}

	if monster, err := loader.LoadMonster(target); err == nil {
		mod := data.CalculateModifier(monster.Dexterity)
		return TargetRes{Type: "Monster", Name: monster.Name, InitiativeMod: mod}, nil
	}

	return TargetRes{}, fmt.Errorf("entity tracking failed: could not locate %s as either Character or Monster in data files", target)
}
