package rules

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Registry manages the CEL environment and provides helper methods for evaluation.
type Registry struct {
	env *cel.Env
}

// NewRegistry initializes the CEL environment with RPG-specific variables and functions.
func NewRegistry(rollFunc func(string) int) (*Registry, error) {
	env, err := cel.NewEnv(
		// Variable declarations
		cel.Variable("actor", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("target", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("action", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("globals", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("roll_result", cel.IntType),

		// Custom RPG functions
		cel.Function("roll",
			cel.Overload("roll_string",
				[]*cel.Type{cel.StringType},
				cel.IntType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					s := arg.Value().(string)
					return types.Int(rollFunc(s))
				}),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &Registry{env: env}, nil
}

// Eval executes a CEL expression against the provided context.
func (r *Registry) Eval(expression string, context map[string]any) (any, error) {
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
