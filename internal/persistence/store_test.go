package persistence

import (
	"path/filepath"
	"testing"

	"github.com/suderio/ancient-draconic/internal/engine"
)

func TestStoreAppendLoad(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "log.jsonl")

	store, err := NewStore(logPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.Append(&engine.ActorAddedEvent{
		ID:    "goblin1",
		Name:  "Test Goblin",
		MaxHP: 10,
	})
	if err != nil {
		t.Fatalf("failed to append actor added: %v", err)
	}

	err = store.Append(&engine.AttributeChangedEvent{
		ActorID:  "goblin1",
		AttrType: engine.AttrSpent,
		Key:      "hp",
		Value:    3,
	})
	if err != nil {
		t.Fatalf("failed to append attribute changed: %v", err)
	}

	// Read it back
	events, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events loaded, got %d", len(events))
	}

	// Verify Event Type Casting works properly
	e1, ok := events[0].(*engine.ActorAddedEvent)
	if !ok {
		t.Errorf("expected first event to be ActorAddedEvent")
	} else if e1.ID != "goblin1" {
		t.Errorf("expected ID goblin1, got %s", e1.ID)
	}

	e2, ok := events[1].(*engine.AttributeChangedEvent)
	if !ok {
		t.Errorf("expected second event to be AttributeChangedEvent")
	} else if val, ok := e2.Value.(float64); !ok || int(val) != 3 {
		// JSON parses numbers into interface{} as float64
		t.Errorf("expected value 3, got %v", e2.Value)
	}
}
