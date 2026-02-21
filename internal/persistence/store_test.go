package persistence

import (
	"path/filepath"
	"testing"

	"github.com/suderio/dndsl/internal/engine"
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

	err = store.Append(&engine.HPChangedEvent{
		ActorID: "goblin1",
		Amount:  -3,
	})
	if err != nil {
		t.Fatalf("failed to append hp changed: %v", err)
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

	e2, ok := events[1].(*engine.HPChangedEvent)
	if !ok {
		t.Errorf("expected second event to be HPChangedEvent")
	} else if e2.Amount != -3 {
		t.Errorf("expected amount -3, got %d", e2.Amount)
	}
}
