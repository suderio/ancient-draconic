package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCELRegistry(t *testing.T) {
	// Mock roll function that returns a fixed value for testing
	mockRoll := func(s string) int {
		if s == "1d20" {
			return 15
		}
		return 0
	}

	registry, err := NewRegistry(nil, mockRoll, nil)
	assert.NoError(t, err)

	t.Run("Basic Boolean Expression", func(t *testing.T) {
		ctx := map[string]any{
			"actor": map[string]any{"dex": 16},
		}
		out, err := registry.Eval("actor.dex > 10", ctx)
		assert.NoError(t, err)
		assert.Equal(t, true, out)
	})

	t.Run("Custom Roll Function", func(t *testing.T) {
		ctx := map[string]any{}
		out, err := registry.Eval("roll('1d20')", ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(15), out) // CEL returns int64 for IntType
	})

	t.Run("Complex Rule", func(t *testing.T) {
		ctx := map[string]any{
			"actor":  map[string]any{"class": "Rogue"},
			"target": map[string]any{"is_flanked": true},
		}
		// Sneak Attack condition example
		expr := "actor.class == 'Rogue' && target.is_flanked"
		out, err := registry.Eval(expr, ctx)
		assert.NoError(t, err)
		assert.Equal(t, true, out)
	})

	t.Run("Global Constants", func(t *testing.T) {
		ctx := map[string]any{
			"globals": map[string]any{"gravity": 1.0},
		}
		out, err := registry.Eval("globals.gravity < 2.0", ctx)
		assert.NoError(t, err)
		assert.Equal(t, true, out)
	})
}
