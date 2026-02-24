package command

import (
	"fmt"
	"strings"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/parser"
)

// Result encapsulation for tracking target types
type TargetRes struct {
	Category      string // 'Character' | 'Monster'
	EntityType    string // Genre-specific: 'undead', etc.
	Name          string
	HP            int
	Stats         map[string]int
	Abilities     []data.Ability
	Proficiencies map[string]int
	Defenses      []data.Defense
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
		profs := make(map[string]int)
		for _, p := range char.Proficiencies {
			profs[p.Proficiency.Index] = p.Value
		}
		return TargetRes{
			Category:      "Character",
			EntityType:    char.Race, // Mapping race as type for characters
			Name:          char.Name,
			HP:            char.HitPoints,
			Stats:         char.GetStats(),
			Abilities:     char.GetAbilities(),
			Proficiencies: profs,
			Defenses:      char.Defenses,
			InitiativeMod: mod,
		}, nil
	}

	if monster, err := loader.LoadMonster(target); err == nil {
		mod := data.CalculateModifier(monster.Dexterity)
		profs := make(map[string]int)
		for _, p := range monster.Proficiencies {
			profs[p.Proficiency.Index] = p.Value
		}
		return TargetRes{
			Category:      "Monster",
			EntityType:    monster.Type,
			Name:          monster.Name,
			HP:            monster.HitPoints,
			Stats:         monster.GetStats(),
			Abilities:     monster.GetAbilities(),
			Proficiencies: profs,
			Defenses:      monster.Defenses,
			InitiativeMod: mod,
		}, nil
	}

	return TargetRes{}, fmt.Errorf("entity tracking failed: could not locate %s as either Character or Monster in data files", target)
}
