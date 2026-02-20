package data

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

// Action represents an action a monster can take
type Action struct {
	Name        string   `yaml:"name"`
	Desc        string   `yaml:"desc"`
	AttackBonus int      `yaml:"attack_bonus"`
	Damage      []Damage `yaml:"damage"`
}

// ArmorClass represents the varying AC calculation rules
type ArmorClass struct {
	Type  string `yaml:"type"`
	Value int    `yaml:"value"`
}

// Monster represents a basic monster from the SRD loaded via YAML.
type Monster struct {
	Index        string            `yaml:"index"`
	Name         string            `yaml:"name"`
	Size         string            `yaml:"size"`
	Type         string            `yaml:"type"`
	Alignment    string            `yaml:"alignment"`
	ArmorClass   []ArmorClass      `yaml:"armor_class"`
	HitPoints    int               `yaml:"hit_points"`
	HitDice      string            `yaml:"hit_dice"`
	Speed        map[string]string `yaml:"speed"`
	Strength     int               `yaml:"strength"`
	Dexterity    int               `yaml:"dexterity"`
	Constitution int               `yaml:"constitution"`
	Intelligence int               `yaml:"intelligence"`
	Wisdom       int               `yaml:"wisdom"`
	Charisma     int               `yaml:"charisma"`
	Actions      []Action          `yaml:"actions"`
}

// Character represents a player character from the data loaded via YAML.
type Character struct {
	Index        string       `yaml:"index"`
	Name         string       `yaml:"name"`
	HitPoints    int          `yaml:"hit_points"`
	HitDice      string       `yaml:"hit_dice"`
	ArmorClass   []ArmorClass `yaml:"armor_class"`
	Strength     int          `yaml:"strength"`
	Dexterity    int          `yaml:"dexterity"`
	Constitution int          `yaml:"constitution"`
	Intelligence int          `yaml:"intelligence"`
	Wisdom       int          `yaml:"wisdom"`
	Charisma     int          `yaml:"charisma"`
	Actions      []Action     `yaml:"actions"`
}

// CalculateModifier returns the standard D&D 5e ability modifier for a given score.
func CalculateModifier(score int) int {
	return (score - 10) / 2
}
