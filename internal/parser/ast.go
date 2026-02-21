package parser

import (
	"strings"
)

// Command represents a top-level action inputted into the DSL
type Command struct {
	Roll       *RollCmd       `parser:"( @@"`
	Encounter  *EncounterCmd  `parser:"| @@"`
	Add        *AddCmd        `parser:"| @@"`
	Initiative *InitiativeCmd `parser:"| @@"`
	Attack     *AttackCmd     `parser:"| @@"`
	Damage     *DamageCmd     `parser:"| @@"`
	Turn       *TurnCmd       `parser:"| @@"`
	Hint       *HintCmd       `parser:"| @@"`
	Ask        *AskCmd        `parser:"| @@"`
	Check      *CheckCmd      `parser:"| @@"`
	Help       *HelpCmd       `parser:"| @@ )"`
}

// RollCmd calculates a set of dice expressions
type RollCmd struct {
	Keyword string     `parser:"@(\"roll\"|\"Roll\"|\"ROLL\")"`
	Actor   *ActorExpr `parser:"@@?"`
	Dice    *DiceExpr  `parser:"@@"`
}

// ActorExpr maps parsing the optional ":by Someone" block
type ActorExpr struct {
	Keyword string `parser:"@(\"by\" \":\")"`
	Name    string `parser:"@Ident"`
}

// DiceExpr represents a complex RPG-style dice roll: NdS[k|d h|l Z][a|d][+/-M]
type DiceExpr struct {
	Raw string `parser:"@DiceMacro"`
}

// IsAdvantage is a helper recognizing shorthand Advantage syntax.
func (d *DiceExpr) IsAdvantage() bool {
	return strings.Contains(strings.ToLower(d.Raw), "a")
}

// IsDisadvantage is a helper recognizing shorthand Disadvantage syntax.
func (d *DiceExpr) IsDisadvantage() bool {
	return strings.Contains(strings.ToLower(d.Raw), "d") && !strings.Contains(strings.ToLower(d.Raw), "d6") && !strings.Contains(strings.ToLower(d.Raw), "d8") && !strings.Contains(strings.ToLower(d.Raw), "d1") && !strings.Contains(strings.ToLower(d.Raw), "d2") // Quick hack to prevent catching the 'd' in 2d20
}

// EncounterCmd manages start and ending of initiative tracking
type EncounterCmd struct {
	Keyword string     `parser:"@(\"encounter\"|\"Encounter\"|\"ENCOUNTER\")"`
	Actor   *ActorExpr `parser:"@@?"` // MUST be GM under execution rules, but parsing we catch anyone
	Action  string     `parser:"@(\"start\"|\"end\")"`
	Targets []string   `parser:"( \"with\" \":\" @Ident ( \"and\" \":\" @Ident )* )?"`
}

// AddCmd brings a new actor into an active encounter
type AddCmd struct {
	Keyword string     `parser:"@(\"add\"|\"Add\"|\"ADD\")"`
	Actor   *ActorExpr `parser:"@@?"` // MUST be GM under execution rules
	Targets []string   `parser:"@Ident ( \"and\" \":\" @Ident )*"`
}

// InitiativeCmd logs or rolls initiative for a character/monster
type InitiativeCmd struct {
	Keyword string     `parser:"@(\"initiative\"|\"Initiative\"|\"INITIATIVE\")"`
	Actor   *ActorExpr `parser:"@@?"`   // Target character taking the roll
	Value   *int       `parser:"@Int?"` // Optional manual override, e.g. "initiative :by Paulo 18"
}

// AttackCmd attempts to strike target(s) with a weapon
type AttackCmd struct {
	Keyword string     `parser:"@(\"attack\"|\"Attack\"|\"ATTACK\")"`
	Actor   *ActorExpr `parser:"@@?"`
	Weapon  string     `parser:"\"with\" \":\" @Ident"`
	Targets []string   `parser:"\"to\" \":\" @Ident ( \"and\" \":\" @Ident )*"`
	Dice    *DiceExpr  `parser:"( \"dice\" \":\" @@ )?"`
}

// DamageRollExpr maps an individual damage dice group and its type
type DamageRollExpr struct {
	Dice *DiceExpr `parser:"\"dice\" \":\" @@"`
	Type string    `parser:"( \"type\" \":\" @Ident )?"`
}

// DamageCmd resolves HP reduction after a successful strike
type DamageCmd struct {
	Keyword string            `parser:"@(\"damage\"|\"Damage\"|\"DAMAGE\")"`
	Actor   *ActorExpr        `parser:"@@?"`
	Weapon  string            `parser:"( \"with\" \":\" @Ident )?"`
	Rolls   []*DamageRollExpr `parser:"@@*"`
}

// TurnCmd advances the initiative rotation
type TurnCmd struct {
	Keyword string     `parser:"@(\"turn\"|\"Turn\"|\"TURN\")"`
	Actor   *ActorExpr `parser:"@@?"`
}

// HintCmd queries the game state to explain blockers or current turn
type HintCmd struct {
	Keyword string `parser:"@(\"hint\"|\"Hint\"|\"HINT\")"`
}

// AskCmd requests an environmental or conditional check from an entity
type AskCmd struct {
	Keyword  string          `parser:"@(\"ask\"|\"Ask\"|\"ASK\")"`
	Actor    *ActorExpr      `parser:"@@?"`
	Check    []string        `parser:"\"check\" \":\" @Ident (@Ident)*"`
	Targets  []string        `parser:"\"of\" \":\" @Ident ( \"and\" \":\" @Ident )*"`
	DC       int             `parser:"\"dc\" \":\" @Int"`
	Fails    *AskConsequence `parser:"( \"fails\" \":\" @@ )?"`
	Succeeds *AskConsequence `parser:"( \"succeeds\" \":\" @@ )?"`
}

// AskConsequence defines the mechanical impact of standard rolls
type AskConsequence struct {
	IsDamage   string    `parser:"( @(\"damage\"|\"Damage\"|\"DAMAGE\")"`
	DamageDice *DiceExpr `parser:"  @@ )"`
	HalfDamage bool      `parser:"| @(\"half\"|\"Half\"|\"HALF\")"`
	Condition  string    `parser:"| @Ident"`
}

// CheckCmd executes an asked check or a standalone check
type CheckCmd struct {
	Keyword string     `parser:"@(\"check\"|\"Check\"|\"CHECK\")"`
	Actor   *ActorExpr `parser:"@@?"`
	Check   []string   `parser:"@Ident (@Ident)*"`
}

// HelpCmd provides context-aware guidance
type HelpCmd struct {
	Keyword string     `parser:"@(\"help\"|\"Help\"|\"HELP\")"`
	Actor   *ActorExpr `parser:"@@?"`
	Command string     `parser:"(@Ident|@(\"all\"))?"`
}
