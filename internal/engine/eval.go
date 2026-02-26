package engine

import (
	"fmt"
	"math/rand"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
)

// RollFunc is a function that evaluates a dice expression (e.g., "1d20") and returns the total.
// It is injected to allow deterministic testing.
type RollFunc func(dice string) int

// Evaluator wraps a CEL environment configured for manifest formula evaluation.
type Evaluator struct {
	env      *cel.Env
	rollFunc RollFunc
}

// NewEvaluator creates a CEL environment with all variables and functions needed
// for manifest formula evaluation.
func NewEvaluator(rollFunc RollFunc) (*Evaluator, error) {
	if rollFunc == nil {
		rollFunc = defaultRoll
	}

	env, err := cel.NewEnv(
		ext.Strings(),
		ext.Lists(),

		// Variables available in all formulas
		cel.Variable("actor", cel.DynType),
		cel.Variable("target", cel.DynType),
		cel.Variable("command", cel.DynType),
		cel.Variable("steps", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("metadata", cel.DynType),

		// Custom RPG functions
		cel.Function("roll",
			cel.Overload("roll_string",
				[]*cel.Type{cel.StringType},
				cel.IntType,
				cel.UnaryBinding(func(val ref.Val) ref.Val {
					dice := val.Value().(string)
					result := rollFunc(dice)
					return types.Int(result)
				}),
			),
		),
		cel.Function("mod",
			cel.Overload("mod_int",
				[]*cel.Type{cel.IntType},
				cel.IntType,
				cel.UnaryBinding(func(val ref.Val) ref.Val {
					score := val.Value().(int64)
					return types.Int((score - 10) / 2)
				}),
			),
		),
		// TODO: The stat function should return the ability score associated with
		// a given skill (e.g., stat("athletics") → actor.stats.str). For now, it
		// just passes through the value, deferring the skill→ability mapping.
		cel.Function("stat",
			cel.Overload("stat_int",
				[]*cel.Type{cel.IntType},
				cel.IntType,
				cel.UnaryBinding(func(val ref.Val) ref.Val {
					// stat() is an alias for an ability score lookup.
					// In formulas like mod(stat(proficiencies[command.skill])),
					// it just passes through the value.
					return val
				}),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Evaluator{env: env, rollFunc: rollFunc}, nil
}

// Eval compiles and evaluates a CEL expression against the given context.
// The context is a map of variable name → value that will be available in the formula.
func (ev *Evaluator) Eval(formula string, ctx map[string]any) (any, error) {
	// Extend the base env with any dynamic variables from the context
	// (e.g., is_encounter_start_active) that aren't pre-declared.
	env, err := ev.extendEnvForContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("CEL env extension error: %w", err)
	}

	ast, issues := env.Compile(formula)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compile error: %w", issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("CEL program error: %w", err)
	}

	out, _, err := prg.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("CEL eval error: %w", err)
	}

	return convertRefVal(out), nil
}

// predeclaredVars is the set of variables declared in the base CEL environment.
var predeclaredVars = map[string]bool{
	"actor": true, "target": true, "command": true, "steps": true, "metadata": true,
}

// extendEnvForContext creates a child CEL environment that includes
// declarations for any dynamic variables in the context (e.g., is_encounter_start_active)
// that are not in the base environment.
func (ev *Evaluator) extendEnvForContext(ctx map[string]any) (*cel.Env, error) {
	var opts []cel.EnvOption
	for key := range ctx {
		if predeclaredVars[key] {
			continue
		}
		opts = append(opts, cel.Variable(key, cel.DynType))
	}
	if len(opts) == 0 {
		return ev.env, nil
	}
	return ev.env.Extend(opts...)
}

// convertRefVal converts a CEL ref.Val to a native Go value, recursively handling
// maps and lists so that downstream code can use standard Go type assertions.
func convertRefVal(val ref.Val) any {
	native := val.Value()
	switch v := native.(type) {
	case map[ref.Val]ref.Val:
		result := make(map[string]any, len(v))
		for mk, mv := range v {
			result[fmt.Sprintf("%v", mk.Value())] = convertRefVal(mv)
		}
		return result
	case []ref.Val:
		result := make([]any, len(v))
		for i, rv := range v {
			result[i] = convertRefVal(rv)
		}
		return result
	default:
		return native
	}
}

// BuildContext constructs the CEL evaluation context from the current game state,
// acting entity, current target, command parameters, and accumulated step results.
func BuildContext(state *GameState, actor *Entity, target *Entity, params map[string]any, stepResults map[string]any, m *Manifest) map[string]any {
	ctx := map[string]any{
		"command":  params,
		"steps":    stepResults,
		"metadata": state.Metadata,
	}

	if actor != nil {
		ctx["actor"] = entityToMap(actor)
	} else {
		ctx["actor"] = map[string]any{}
	}

	if target != nil {
		ctx["target"] = entityToMap(target)
	} else {
		ctx["target"] = map[string]any{}
	}

	// Inject default false for every command's loop variable so CEL never
	// encounters an undeclared reference. Commands use is_<name>_active in prereqs.
	if m != nil {
		for name := range m.Commands {
			ctx["is_"+name+"_active"] = false
		}
	}

	// Override with actual loop state
	for name, loop := range state.Loops {
		ctx["is_"+name+"_active"] = loop.Active
	}

	return ctx
}

// entityToMap converts an Entity to a map[string]any suitable for CEL evaluation.
// This allows formulas to access fields like actor.stats.str, actor.spent.actions, etc.
func entityToMap(e *Entity) map[string]any {
	return map[string]any{
		"id":            e.ID,
		"name":          e.Name,
		"types":         e.Types,
		"classes":       e.Classes,
		"stats":         intMapToAny(e.Stats),
		"resources":     intMapToAny(e.Resources),
		"spent":         intMapToAny(e.Spent),
		"conditions":    e.Conditions,
		"proficiencies": intMapToAny(e.Proficiencies),
		"statuses":      strMapToAny(e.Statuses),
		"inventory":     intMapToAny(e.Inventory),
	}
}

// intMapToAny converts map[string]int to map[string]any so CEL can use dynamic typing.
func intMapToAny(m map[string]int) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = int64(v) // CEL uses int64 for integers
	}
	return result
}

// strMapToAny converts map[string]string to map[string]any so CEL can use dynamic typing.
func strMapToAny(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// defaultRoll provides a basic dice roller for production use.
// For testing, inject a deterministic RollFunc instead.
func defaultRoll(dice string) int {
	// Very basic parser: handles "1d20", "2d6", etc.
	var count, sides int
	if _, err := fmt.Sscanf(dice, "%dd%d", &count, &sides); err != nil || sides <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < count; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}
