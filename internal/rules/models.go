package rules

// DiceRoller is an interface for something that can roll dice.
type DiceRoller interface {
	Roll(expr string) int
}

// GlobalContext represents the root object passed to CEL evaluation.
// This allows us to expose multiple objects under a single root if needed,
// but usually we bind them as individual variables.
type GlobalContext struct {
	Actor      map[string]any
	Target     map[string]any
	Action     map[string]any
	Globals    map[string]any
	RollResult int
}
