package parser

import (
	"strings"
)

// Command represents a top-level action inputted into the DSL
type Command struct {
	Roll       *RollCmd       `parser:"( @@"`
	Encounter  *EncounterCmd  `parser:"| @@"`
	Add        *AddCmd        `parser:"| @@"`
	Hint       *HintCmd       `parser:"| @@"`
	Adjudicate *AdjudicateCmd `parser:"| @@"`
	Allow      *AllowCmd      `parser:"| @@"`
	Deny       *DenyCmd       `parser:"| @@"`
	Help       *HelpCmd       `parser:"| @@"`
	Generic    *GenericCmd    `parser:"| @@ )"`
}

// RollCmd calculates a set of dice expressions
type RollCmd struct {
	Keyword string     `parser:"@(\"roll\"|\"Roll\"|\"ROLL\")"`
	Actor   *ActorExpr `parser:"@@?"`
	Dice    *DiceExpr  `parser:"@@"`
}

// ActorExpr maps parsing the optional "by: Someone" block
type ActorExpr struct {
	Keyword string `parser:"\"by\" \":\""`
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
	Targets []string   `parser:"( \"with\" \":\" @(Ident|DiceMacro|Int|String) ( \"and\" \":\" @(Ident|DiceMacro|Int|String) )* )?"`
}

// AddCmd brings a new actor into an active encounter
type AddCmd struct {
	Keyword string     `parser:"@(\"add\"|\"Add\"|\"ADD\")"`
	Actor   *ActorExpr `parser:"@@?"` // MUST be GM under execution rules
	Targets []string   `parser:"@(Ident|DiceMacro|Int|String) ( \"and\" \":\" @(Ident|DiceMacro|Int|String) )*"`
}

// HintCmd queries the game state to explain blockers or current turn
type HintCmd struct {
	Keyword string `parser:"@(\"hint\"|\"Hint\"|\"HINT\")"`
}

// HelpCmd provides context-aware guidance
type HelpCmd struct {
	Keyword string     `parser:"@(\"help\"|\"Help\"|\"HELP\")"`
	Actor   *ActorExpr `parser:"@@?"`
	Command string     `parser:"(@Ident|@(\"all\"))?"`
}

// AdjudicateCmd requests GM authorization
type AdjudicateCmd struct {
	Keyword string `parser:"@(\"adjudicate\"|\"Adjudicate\"|\"ADJUDICATE\")"`
	Command string `parser:"@String"`
}

// AllowCmd approves a pending adjudication
type AllowCmd struct {
	Keyword string     `parser:"@(\"allow\"|\"Allow\"|\"ALLOW\")"`
	Actor   *ActorExpr `parser:"@@?"`
}

// DenyCmd rejects a pending adjudication
type DenyCmd struct {
	Keyword string     `parser:"@(\"deny\"|\"Deny\"|\"DENY\")"`
	Actor   *ActorExpr `parser:"@@?"`
}

// GenericCmd handles ANY arbitrary command routed to the data-driven manifest
type GenericCmd struct {
	Name  string     `parser:"@Ident"`
	Actor *ActorExpr `parser:"@@?"`
	Args  []*ArgExpr `parser:"@@*"`
}

// ArgExpr handles generic key-value tuples (e.g. "with: sword", "to: goblin and: orc")
type ArgExpr struct {
	Key    string   `parser:"@Ident \":\""`
	Values []string `parser:"@(Ident|DiceMacro|Int|String) ( \"and\" \":\" @(Ident|DiceMacro|Int|String) )*"` // e.g. "target1 and: target2"
}
