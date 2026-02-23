package rules

import (
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"github.com/suderio/ancient-draconic/internal/data"
)

// DiceReporter allows the registry to report dice rolls back to the engine
type DiceReporter func(dice string, result int)

// Registry manages the CEL environment and provides helper methods for evaluation.
type Registry struct {
	env          *cel.Env
	manifest     *data.CampaignManifest
	diceReporter DiceReporter
}

// NewRegistry initializes the CEL environment with RPG-specific variables and functions.
func NewRegistry(manifest *data.CampaignManifest, rollFunc func(string) int, reporter DiceReporter) (*Registry, error) {
	r := &Registry{manifest: manifest, diceReporter: reporter}
	env, err := cel.NewEnv(
		ext.Strings(),
		ext.Lists(),
		// Variable declarations
		cel.Variable("actor", cel.DynType),
		cel.Variable("target", cel.DynType),
		cel.Variable("action", cel.DynType),
		cel.Variable("globals", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("profs", cel.ListType(cel.StringType)),
		cel.Variable("roll_result", cel.IntType),
		cel.Variable("manifest", cel.DynType),
		cel.Variable("steps", cel.MapType(cel.StringType, cel.AnyType)),

		// Custom RPG functions
		cel.Function("get_condition",
			cel.Overload("get_condition_map_string",
				[]*cel.Type{cel.DynType, cel.StringType},
				cel.StringType,
				cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
					ent, ok := lhs.Value().(map[string]any)
					if !ok {
						return types.String("")
					}
					prefix := rhs.Value().(string)
					conds, ok := ent["conditions"].([]string)
					if !ok {
						return types.String("")
					}
					for _, c := range conds {
						if strings.HasPrefix(c, prefix) {
							return types.String(c)
						}
					}
					return types.String("")
				}),
			),
		),
		cel.Function("roll",
			cel.Overload("roll_string",
				[]*cel.Type{cel.StringType},
				cel.IntType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					s := arg.Value().(string)
					res := rollFunc(s)
					if r.diceReporter != nil {
						r.diceReporter(s, res)
					}
					return types.Int(res)
				}),
			),
		),
		cel.Function("mod",
			cel.Overload("mod_int",
				[]*cel.Type{cel.IntType},
				cel.IntType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					score := int(arg.Value().(int64))
					return types.Int((score - 10) / 2)
				}),
			),
		),
		cel.Function("size_rank",
			cel.Overload("size_rank_string",
				[]*cel.Type{cel.StringType},
				cel.IntType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					s := strings.ToLower(arg.Value().(string))
					switch s {
					case "tiny":
						return types.Int(1)
					case "small":
						return types.Int(2)
					case "medium":
						return types.Int(3)
					case "large":
						return types.Int(4)
					case "huge":
						return types.Int(5)
					case "gargantuan":
						return types.Int(6)
					default:
						return types.Int(0)
					}
				}),
			),
		),
		cel.Function("float",
			cel.Overload("float_int",
				[]*cel.Type{cel.IntType},
				cel.DoubleType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					return types.Double(float64(arg.Value().(int64)))
				}),
			),
		),
		cel.Variable("pending_adjudication", cel.DynType),
		cel.Variable("is_frozen", cel.BoolType),
	)
	if err != nil {
		return nil, err
	}
	r.env = env
	return r, nil
}

// SetDiceReporter updates the reporter for the registry
func (r *Registry) SetDiceReporter(reporter DiceReporter) {
	r.diceReporter = reporter
}

// Eval executes a CEL expression against the provided context.
func (r *Registry) Eval(expression string, context map[string]any) (any, error) {
	// Inject manifest into context if not present
	if _, ok := context["manifest"]; !ok && r.manifest != nil {
		context["manifest"] = map[string]any{
			"system":       r.manifest.System,
			"global_rules": r.manifest.GlobalRules,
		}
	}

	ast, iss := r.env.Compile(expression)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	prog, err := r.env.Program(ast)
	if err != nil {
		return nil, err
	}
	out, _, err := prog.Eval(context)
	if err != nil {
		return nil, err
	}
	return out.Value(), nil
}

// GetCommand returns the definition for a given command from the manifest.
func (r *Registry) GetCommand(name string) (data.CommandDefinition, bool) {
	if r.manifest == nil {
		return data.CommandDefinition{}, false
	}
	cmd, ok := r.manifest.Commands[name]
	return cmd, ok
}
