package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Event Type/Message/Apply tests ---

func TestLoopEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &LoopEvent{LoopName: "combat", Active: true}
	assert.Equal(t, "LoopEvent", evt.Type())
	assert.Equal(t, "combat started", evt.Message())
	require.NoError(t, evt.Apply(state))
	assert.True(t, state.Loops["combat"].Active)

	evt2 := &LoopEvent{LoopName: "combat", Active: false}
	assert.Equal(t, "combat ended", evt2.Message())
	require.NoError(t, evt2.Apply(state))
	assert.False(t, state.Loops["combat"].Active)
}

func TestLoopOrderAscendingEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Loops["combat"] = &Loop{Active: true, Order: make(map[string]int)}

	evt := &LoopOrderAscendingEvent{LoopName: "combat", Ascending: true}
	assert.Equal(t, "LoopOrderAscendingEvent", evt.Type())
	assert.Equal(t, "", evt.Message())
	require.NoError(t, evt.Apply(state))
	assert.True(t, state.Loops["combat"].Ascending)

	// Apply to non-existent loop is a no-op
	evt2 := &LoopOrderAscendingEvent{LoopName: "missing"}
	require.NoError(t, evt2.Apply(state))
}

func TestLoopOrderEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Loops["combat"] = &Loop{Active: true, Order: make(map[string]int)}

	evt := &LoopOrderEvent{LoopName: "combat", ActorID: "fighter", Value: 18}
	assert.Equal(t, "LoopOrderEvent", evt.Type())
	assert.Equal(t, "fighter order set", evt.Message())
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 18, state.Loops["combat"].Order["fighter"])

	// Non-existent loop is a no-op
	evt2 := &LoopOrderEvent{LoopName: "missing", ActorID: "x", Value: 1}
	require.NoError(t, evt2.Apply(state))
}

func TestActorAddedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Loops["combat"] = &Loop{Active: true, Actors: []string{}, Order: make(map[string]int)}

	evt := &ActorAddedEvent{LoopName: "combat", ActorID: "fighter"}
	assert.Equal(t, "ActorAddedEvent", evt.Type())
	assert.Equal(t, "fighter added to combat", evt.Message())
	require.NoError(t, evt.Apply(state))
	assert.Contains(t, state.Loops["combat"].Actors, "fighter")

	// Duplicate add is idempotent
	require.NoError(t, evt.Apply(state))
	assert.Len(t, state.Loops["combat"].Actors, 1)

	// Non-existent loop is a no-op
	evt2 := &ActorAddedEvent{LoopName: "missing", ActorID: "x"}
	require.NoError(t, evt2.Apply(state))
}

func TestAttributeChangedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Entities["fighter"] = NewEntity("fighter", "Fighter")

	tests := []struct {
		section string
		key     string
		value   any
	}{
		{"stats", "str", 18},
		{"resources", "hp", 45},
		{"spent", "spell_slot", 2},
		{"statuses", "poisoned", "yes"},
		{"classes", "level", "5"},
		{"inventory", "potion", 3},
	}

	for _, tt := range tests {
		evt := &AttributeChangedEvent{ActorID: "fighter", Section: tt.section, Key: tt.key, Value: tt.value}
		assert.Equal(t, "AttributeChangedEvent", evt.Type())
		assert.Contains(t, evt.Message(), tt.key)
		require.NoError(t, evt.Apply(state))
	}

	// Unknown section — Apply silently ignores it
	evt := &AttributeChangedEvent{ActorID: "fighter", Section: "unknown", Key: "x", Value: 1}
	err := evt.Apply(state)
	assert.NoError(t, err)

	// Unknown actor
	evt2 := &AttributeChangedEvent{ActorID: "nobody", Section: "stats", Key: "str", Value: 10}
	err2 := evt2.Apply(state)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "not found")
}

func TestConditionEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Entities["fighter"] = NewEntity("fighter", "Fighter")

	// Add condition
	evt := &ConditionEvent{ActorID: "fighter", Condition: "poisoned", Add: true}
	assert.Equal(t, "ConditionEvent", evt.Type())
	assert.Contains(t, evt.Message(), "poisoned")
	require.NoError(t, evt.Apply(state))
	assert.Contains(t, state.Entities["fighter"].Conditions, "poisoned")

	// Remove condition
	evt2 := &ConditionEvent{ActorID: "fighter", Condition: "poisoned", Add: false}
	assert.Contains(t, evt2.Message(), "poisoned")
	require.NoError(t, evt2.Apply(state))
	assert.NotContains(t, state.Entities["fighter"].Conditions, "poisoned")

	// Remove non-existent condition is a no-op
	require.NoError(t, evt2.Apply(state))

	// Unknown actor
	evt3 := &ConditionEvent{ActorID: "nobody", Condition: "x", Add: true}
	err := evt3.Apply(state)
	assert.Error(t, err)
}

func TestAddSpentEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Entities["fighter"] = NewEntity("fighter", "Fighter")

	evt := &AddSpentEvent{ActorID: "fighter", Key: "spell_slot"}
	assert.Equal(t, "AddSpentEvent", evt.Type())
	assert.Contains(t, evt.Message(), "spell_slot")
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 1, state.Entities["fighter"].Spent["spell_slot"])

	// Increment again
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 2, state.Entities["fighter"].Spent["spell_slot"])

	// Unknown actor
	evt2 := &AddSpentEvent{ActorID: "nobody", Key: "x"}
	err := evt2.Apply(state)
	assert.Error(t, err)
}

func TestDiceRolledEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &DiceRolledEvent{ActorID: "fighter", Dice: "2d6+3", Result: 12}
	assert.Equal(t, "DiceRolledEvent", evt.Type())
	assert.Contains(t, evt.Message(), "2d6+3")
	assert.Contains(t, evt.Message(), "12")
	// Apply is a no-op for display-only events
	require.NoError(t, evt.Apply(state))
}

func TestAskIssuedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &AskIssuedEvent{TargetID: "player1", Options: []string{"attack", "defend"}}
	assert.Equal(t, "AskIssuedEvent", evt.Type())
	assert.Contains(t, evt.Message(), "player1")
	require.NoError(t, evt.Apply(state))
	// Apply sets pending_ask in metadata
	assert.NotNil(t, state.Metadata["pending_ask"])
}

func TestMetadataChangedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &MetadataChangedEvent{Key: "round", Value: 3}
	assert.Equal(t, "MetadataChangedEvent", evt.Type())
	assert.Contains(t, evt.Message(), "round")
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 3, state.Metadata["round"])
}

func TestCheckEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &CheckEvent{ActorID: "fighter", Check: "athletics", Passed: true}
	assert.Equal(t, "CheckEvent", evt.Type())
	assert.Contains(t, evt.Message(), "athletics")
	assert.Contains(t, evt.Message(), "passed")
	require.NoError(t, evt.Apply(state))
	assert.NotNil(t, state.Metadata["last_check"])

	evt2 := &CheckEvent{ActorID: "wizard", Check: "arcana", Passed: false}
	assert.Contains(t, evt2.Message(), "failed")
}

func TestHintEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &HintEvent{MessageStr: "Remember to use bonus action"}
	assert.Equal(t, "HintEvent", evt.Type())
	assert.Equal(t, "Remember to use bonus action", evt.Message())
	require.NoError(t, evt.Apply(state))
}

func TestCustomEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &CustomEvent{EventType: "fire_bolt", ActorID: "wizard", Payload: map[string]any{"damage": 42}}
	assert.Equal(t, "CustomEvent", evt.Type())
	assert.Contains(t, evt.Message(), "fire_bolt")
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, map[string]any{"damage": 42}, state.Metadata["fire_bolt"])
}

func TestTurnEndedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()

	evt := &TurnEndedEvent{LoopName: "combat", ActorID: "fighter"}
	assert.Equal(t, "TurnEndedEvent", evt.Type())
	assert.Contains(t, evt.Message(), "fighter")
	require.NoError(t, evt.Apply(state))
}

func TestTurnStartedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Loops["combat"] = &Loop{
		Active: true,
		Actors: []string{"fighter", "wizard"},
		Order:  map[string]int{"fighter": 18, "wizard": 12},
	}

	evt := &TurnStartedEvent{LoopName: "combat", ActorID: "wizard", Turn: 2}
	assert.Equal(t, "TurnStartedEvent", evt.Type())
	assert.Contains(t, evt.Message(), "wizard")
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 2, state.Loops["combat"].Turn)
	assert.Equal(t, 1, state.Loops["combat"].Current) // wizard is at index 1
}

func TestRoundStartedEvent_TypeMessageApply(t *testing.T) {
	state := NewGameState()
	state.Loops["combat"] = &Loop{Active: true, Order: make(map[string]int)}

	evt := &RoundStartedEvent{LoopName: "combat", Round: 3}
	assert.Equal(t, "RoundStartedEvent", evt.Type())
	assert.Contains(t, evt.Message(), "round 3")
	require.NoError(t, evt.Apply(state))
	assert.Equal(t, 3, state.Loops["combat"].Round)

	// Non-existent loop
	evt2 := &RoundStartedEvent{LoopName: "missing", Round: 1}
	require.NoError(t, evt2.Apply(state))
}

// --- IsLoopActive edge case ---
func TestIsLoopActive_NoLoop(t *testing.T) {
	state := NewGameState()
	assert.False(t, state.IsLoopActive("nonexistent"))
}

// --- toInt tests ---
func TestToInt(t *testing.T) {
	tests := []struct {
		input    any
		expected int
		ok       bool
	}{
		{42, 42, true},
		{int64(99), 99, true},
		{float64(3.14), 3, true},
		{"hello", 0, false},
		{nil, 0, false},
		{true, 0, false},
	}
	for _, tt := range tests {
		v, ok := toInt(tt.input)
		assert.Equal(t, tt.ok, ok, "input: %v", tt.input)
		if ok {
			assert.Equal(t, tt.expected, v, "input: %v", tt.input)
		}
	}
}

// --- NewEntity ---
func TestNewEntity(t *testing.T) {
	e := NewEntity("test", "Test")
	assert.NotNil(t, e.Types)
	assert.NotNil(t, e.Classes)
	assert.NotNil(t, e.Stats)
	assert.NotNil(t, e.Resources)
	assert.NotNil(t, e.Spent)
	assert.NotNil(t, e.Conditions)
	assert.NotNil(t, e.Proficiencies)
	assert.NotNil(t, e.Statuses)
	assert.NotNil(t, e.Inventory)
}
