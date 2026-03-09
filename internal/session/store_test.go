package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suderio/ancient-draconic/internal/engine"
)

func TestStoreAppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	store, err := NewStore(path)
	require.NoError(t, err)

	evt := &engine.LoopEvent{LoopName: "combat", Active: true}
	require.NoError(t, store.Append(evt))

	events, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "LoopEvent", events[0].Type())

	store.Close()
}

func TestStoreTruncate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	store, err := NewStore(path)
	require.NoError(t, err)

	// Add 5 events
	for i := 0; i < 5; i++ {
		require.NoError(t, store.Append(&engine.HintEvent{MessageStr: "msg"}))
	}

	count, err := store.EventCount()
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Truncate to 3
	require.NoError(t, store.Truncate(3))

	count, err = store.EventCount()
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Truncate to 0
	require.NoError(t, store.Truncate(0))

	count, err = store.EventCount()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	store.Close()
}

func TestStoreTruncate_KeepMoreThanExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	store, err := NewStore(path)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Append(&engine.HintEvent{MessageStr: "msg"}))

	// keepN > actual count should not panic
	require.NoError(t, store.Truncate(100))

	count, err := store.EventCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStoreTruncate_NegativeKeepN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	store, err := NewStore(path)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Append(&engine.HintEvent{MessageStr: "msg"}))

	// Negative keepN should truncate to 0
	require.NoError(t, store.Truncate(-5))

	count, err := store.EventCount()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestStoreLoad_AllEventTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	store, err := NewStore(path)
	require.NoError(t, err)

	events := []engine.Event{
		&engine.LoopEvent{LoopName: "combat", Active: true},
		&engine.LoopOrderAscendingEvent{LoopName: "combat", Ascending: false},
		&engine.LoopOrderEvent{LoopName: "combat", ActorID: "fighter", Value: 18},
		&engine.ActorAddedEvent{LoopName: "combat", ActorID: "fighter"},
		&engine.AttributeChangedEvent{ActorID: "fighter", Section: "stats", Key: "str", Value: 18},
		&engine.ConditionEvent{ActorID: "fighter", Condition: "poisoned", Add: true},
		&engine.AddSpentEvent{ActorID: "fighter", Key: "slot"},
		&engine.DiceRolledEvent{ActorID: "fighter", Dice: "1d20", Result: 15},
		&engine.AskIssuedEvent{TargetID: "player1", Options: []string{"a", "b"}},
		&engine.MetadataChangedEvent{Key: "round", Value: 1},
		&engine.CheckEvent{ActorID: "fighter", Check: "athletics", Passed: true},
		&engine.HintEvent{MessageStr: "hint"},
		&engine.CustomEvent{EventType: "custom", ActorID: "wizard", Payload: map[string]any{"x": 1}},
		&engine.TurnEndedEvent{LoopName: "combat", ActorID: "fighter"},
		&engine.TurnStartedEvent{LoopName: "combat", ActorID: "wizard", Turn: 2},
		&engine.RoundStartedEvent{LoopName: "combat", Round: 1},
	}

	for _, evt := range events {
		require.NoError(t, store.Append(evt))
	}
	store.Close()

	// Reload and verify all events round-trip
	store2, err := NewStore(path)
	require.NoError(t, err)
	defer store2.Close()

	loaded, err := store2.Load()
	require.NoError(t, err)
	assert.Len(t, loaded, len(events))

	for i, evt := range loaded {
		assert.Equal(t, events[i].Type(), evt.Type(), "event %d type mismatch", i)
	}
}

func TestStoreLoad_UnknownEventType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Write a raw JSONL line with an unknown event type
	err := os.WriteFile(path, []byte(`{"type":"UnknownFutureEvent","data":{}}`+"\n"), 0644)
	require.NoError(t, err)

	store, err := NewStore(path)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestStoreLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	err := os.WriteFile(path, []byte("not json\n"), 0644)
	require.NoError(t, err)

	store, err := NewStore(path)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.Load()
	assert.Error(t, err)
}

func TestNewStore_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.jsonl")

	store, err := NewStore(path)
	require.NoError(t, err)
	defer store.Close()

	_, err = os.Stat(path)
	assert.NoError(t, err)
}
