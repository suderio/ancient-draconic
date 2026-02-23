package data

// Ability represents a data-driven rule or feature
type Ability struct {
	Name      string `json:"name" yaml:"name"`
	Condition string `json:"condition" yaml:"condition"` // CEL expression
	Effect    string `json:"effect" yaml:"effect"`       // CEL expression or descriptive impact
}

// CommandStep defines a single evaluation step in a command chain
type CommandStep struct {
	Name    string `json:"name" yaml:"name"`
	Formula string `json:"formula" yaml:"formula"` // CEL expression
	Event   string `json:"event" yaml:"event"`     // Mapping to engine.Event type
}

// CommandDefinition defines how a generic command (e.g., 'attack') behaves
type CommandDefinition struct {
	Name  string        `json:"name" yaml:"name"`
	Steps []CommandStep `json:"steps" yaml:"steps"`
}

// CampaignManifest centralizes all system-specific rules and command definitions
type CampaignManifest struct {
	System      string                       `json:"system" yaml:"system"`
	Commands    map[string]CommandDefinition `json:"commands" yaml:"commands"`
	GlobalRules map[string]string            `json:"global_rules" yaml:"global_rules"`
}

// Reference represents a standard 5e API reference pointer (e.g. index: 'force', name: 'Force', ref: 'damage-types/force.yaml')
type Reference struct {
	Index string `yaml:"index"`
	Name  string `yaml:"name"`
	Ref   string `yaml:"ref"`
}

// Damage represents damage applied in DnD (e.g. 1d6 + 2 slashing)
type Damage struct {
	DamageDice string    `yaml:"damage_dice"`
	DamageType Reference `yaml:"damage_type"`
}

// Proficiency defines character trained abilities or saves
type Proficiency struct {
	Proficiency Reference `yaml:"proficiency"`
	Value       int       `yaml:"value"`
}

// Action represents an action a monster can take
type Action struct {
	Name        string   `yaml:"name"`
	Desc        string   `yaml:"desc"`
	AttackBonus int      `yaml:"attack_bonus"`
	HitRule     string   `yaml:"hit_rule" json:"hit_rule"` // CEL formula for hit resolution
	Damage      []Damage `yaml:"damage"`
	Recharge    string   `yaml:"recharge"`
}

// Defense defines a creature's damage modifiers
type Defense struct {
	Resistances     []string `yaml:"resistances"`
	Immunities      []string `yaml:"immunities"`
	Vulnerabilities []string `yaml:"vulnerabilities"`
}

// ArmorClass represents the varying AC calculation rules
type ArmorClass struct {
	Type  string `yaml:"type"`
	Value int    `yaml:"value"`
}

// Monster represents a basic monster from the SRD loaded via YAML.
type Monster struct {
	Index            string            `yaml:"index"`
	Name             string            `yaml:"name"`
	Size             string            `yaml:"size"`
	Type             string            `yaml:"type"`
	Alignment        string            `yaml:"alignment"`
	ArmorClass       []ArmorClass      `yaml:"armor_class"`
	HitPoints        int               `yaml:"hit_points"`
	HitDice          string            `yaml:"hit_dice"`
	Speed            map[string]string `yaml:"speed"`
	Strength         int               `yaml:"strength"`
	Dexterity        int               `yaml:"dexterity"`
	Constitution     int               `yaml:"constitution"`
	Intelligence     int               `yaml:"intelligence"`
	Wisdom           int               `yaml:"wisdom"`
	Charisma         int               `yaml:"charisma"`
	ProficiencyBonus int               `yaml:"proficiency_bonus"`
	Actions          []Action          `yaml:"actions"`
	Proficiencies    []Proficiency     `yaml:"proficiencies"`
	Defenses         []Defense         `yaml:"defenses"`
	SpecialAbilities []Ability         `yaml:"special_abilities"`
}

func (m *Monster) GetStats() map[string]int {
	stats := map[string]int{
		"str":        m.Strength,
		"dex":        m.Dexterity,
		"con":        m.Constitution,
		"int":        m.Intelligence,
		"wis":        m.Wisdom,
		"cha":        m.Charisma,
		"prof_bonus": m.ProficiencyBonus,
	}
	if len(m.ArmorClass) > 0 {
		stats["ac"] = m.ArmorClass[0].Value
	}
	return stats
}

func (m *Monster) GetAbilities() []Ability {
	return m.SpecialAbilities
}

// Race represents a character race from the SRD.
type Race struct {
	Index string `yaml:"index"`
	Name  string `yaml:"name"`
	Size  string `yaml:"size"`
}

// Character represents a player character from the data loaded via YAML.
type Character struct {
	Index            string        `yaml:"index"`
	Name             string        `yaml:"name"`
	Race             string        `yaml:"race"`
	HitPoints        int           `yaml:"hit_points"`
	HitDice          string        `yaml:"hit_dice"`
	ArmorClass       []ArmorClass  `yaml:"armor_class"`
	Strength         int           `yaml:"strength"`
	Dexterity        int           `yaml:"dexterity"`
	Constitution     int           `yaml:"constitution"`
	Intelligence     int           `yaml:"intelligence"`
	Wisdom           int           `yaml:"wisdom"`
	Charisma         int           `yaml:"charisma"`
	ProficiencyBonus int           `yaml:"proficiency_bonus"`
	Actions          []Action      `yaml:"actions"`
	Proficiencies    []Proficiency `yaml:"proficiencies"`
	Defenses         []Defense     `yaml:"defenses"`
	Abilities        []Ability     `yaml:"abilities"`
}

func (c *Character) GetStats() map[string]int {
	stats := map[string]int{
		"str":        c.Strength,
		"dex":        c.Dexterity,
		"con":        c.Constitution,
		"int":        c.Intelligence,
		"wis":        c.Wisdom,
		"cha":        c.Charisma,
		"prof_bonus": c.ProficiencyBonus,
	}
	if len(c.ArmorClass) > 0 {
		stats["ac"] = c.ArmorClass[0].Value
	}
	return stats
}

func (c *Character) GetAbilities() []Ability {
	return c.Abilities
}

// CalculateModifier returns the standard D&D 5e ability modifier for a given score.
func CalculateModifier(score int) int {
	return (score - 10) / 2
}
