package engine

import (
	"testing"
)

func TestProjectorBuild(t *testing.T) {
	events := []Event{
		&EncounterStartedEvent{},
		&ActorAddedEvent{ID: "goblin1", Name: "Goblin", Resources: map[string]int{"hp": 15}},
		&ActorAddedEvent{ID: "fighter1", Name: "Fighter", Resources: map[string]int{"hp": 30}},
		&HPChangedEvent{ActorID: "goblin1", Amount: -5},
		&HPChangedEvent{ActorID: "fighter1", Amount: -10},
		&HPChangedEvent{ActorID: "fighter1", Amount: 2}, // test slight heal
	}

	projector := NewProjector()
	state, err := projector.Build(events)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(state.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(state.Entities))
	}

	goblin := state.Entities["goblin1"]
	if hp := goblin.Resources["hp"] - goblin.Spent["hp"]; hp != 10 {
		t.Errorf("expected goblin HP to be 10, got %d", hp)
	}

	fighter := state.Entities["fighter1"]
	if hp := fighter.Resources["hp"] - fighter.Spent["hp"]; hp != 22 {
		t.Errorf("expected fighter HP to be 22, got %d", hp)
	}
}
