package rules

import (
	"fmt"
	"sort"
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
		cel.Function("min",
			cel.Overload("min_int",
				[]*cel.Type{cel.IntType, cel.IntType},
				cel.IntType,
				cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
					a := lhs.Value().(int64)
					b := rhs.Value().(int64)
					if a < b {
						return lhs
					}
					return rhs
				}),
			),
		),
		cel.Function("max",
			cel.Overload("max_int",
				[]*cel.Type{cel.IntType, cel.IntType},
				cel.IntType,
				cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
					a := lhs.Value().(int64)
					b := rhs.Value().(int64)
					if a > b {
						return lhs
					}
					return rhs
				}),
			),
		),
		cel.Function("sort_initiatives",
			cel.Overload("sort_initiatives_map",
				[]*cel.Type{cel.MapType(cel.StringType, cel.IntType)},
				cel.ListType(cel.StringType),
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					m := arg.Value().(map[string]any)
					initiatives := make(map[string]int)
					var names []string
					for k, v := range m {
						names = append(names, k)
						if i, ok := v.(int64); ok {
							initiatives[k] = int(i)
						} else if f, ok := v.(float64); ok {
							initiatives[k] = int(f)
						} else if i, ok := v.(int); ok {
							initiatives[k] = i
						}
					}

					sort.SliceStable(names, func(i, j int) bool {
						s1 := initiatives[names[i]]
						s2 := initiatives[names[j]]
						if s1 != s2 {
							return s1 > s2
						}
						return names[i] < names[j]
					})

					return types.DefaultTypeAdapter.NativeToValue(names)
				}),
			),
		),
		cel.Function("merge",
			cel.Overload("merge_maps",
				[]*cel.Type{cel.MapType(cel.StringType, cel.AnyType), cel.MapType(cel.StringType, cel.AnyType)},
				cel.MapType(cel.StringType, cel.AnyType),
				cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
					m1 := lhs.Value().(map[string]any)
					m2 := rhs.Value().(map[string]any)
					newMap := make(map[string]any)
					for k, v := range m1 {
						newMap[k] = v
					}
					for k, v := range m2 {
						newMap[k] = v
					}
					return types.DefaultTypeAdapter.NativeToValue(newMap)
				}),
			),
		),
		cel.Variable("pending_adjudication", cel.DynType),
		cel.Variable("is_frozen", cel.BoolType),
		cel.Variable("is_encounter_active", cel.BoolType),
		cel.Variable("pending_checks", cel.DynType),
		cel.Variable("entities", cel.DynType),
		cel.Variable("metadata", cel.MapType(cel.StringType, cel.AnyType)),
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
	if strings.Contains(expression, "immunities") {
	}
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

	result := convertRefVal(out)

	return result, nil
}

func convertRefVal(v ref.Val) any {
	if v == nil {
		return nil
	}
	res := v.Value()

	// Handle maps (including those returned by CEL)
	if m, ok := res.(map[ref.Val]ref.Val); ok {
		nativeMap := make(map[string]any)
		for mk, mv := range m {
			keyStr := fmt.Sprintf("%v", mk.Value())
			nativeMap[keyStr] = convertRefVal(mv)
		}
		return nativeMap
	}

	// Handle lists
	if l, ok := res.([]ref.Val); ok {
		nativeList := make([]any, len(l))
		for i, v := range l {
			nativeList[i] = convertRefVal(v)
		}
		return nativeList
	}

	return res
}

// GetCommand returns the definition for a given command from the manifest.
func (r *Registry) GetCommand(name string) (data.CommandDefinition, bool) {
	if r.manifest == nil {
		return data.CommandDefinition{}, false
	}
	cmd, ok := r.manifest.Commands[name]
	return cmd, ok
}
