package engine

import (
	"fmt"
	"strings"
)

type EventType string

const (
	EventEncounterStarted EventType = "EncounterStarted"
	EventEncounterEnded   EventType = "EncounterEnded"
	EventActorAdded       EventType = "ActorAdded"
	EventTurnChanged      EventType = "TurnChanged"
	EventHPChanged        EventType = "HPChanged"
	EventDiceRolled       EventType = "DiceRolled"
	EventInitiativeRolled EventType = "InitiativeRolled"
)

// Event is the building block of the Event Sourced engine.
type Event interface {
	Type() EventType
	Apply(state *GameState) error
	Message() string
}

// EncounterStartedEvent clears the encounter table.
type EncounterStartedEvent struct{}

func (e *EncounterStartedEvent) Type() EventType { return EventEncounterStarted }
func (e *EncounterStartedEvent) Apply(state *GameState) error {
	state.IsEncounterActive = true
	state.Entities = make(map[string]*Entity)
	state.TurnOrder = make([]string, 0)
	state.Initiatives = make(map[string]int)
	state.CurrentTurn = 0
	return nil
}
func (e *EncounterStartedEvent) Message() string { return "Encounter Started." }

// EncounterEndedEvent drops the active encounter flag
type EncounterEndedEvent struct{}

func (e *EncounterEndedEvent) Type() EventType { return EventEncounterEnded }
func (e *EncounterEndedEvent) Apply(state *GameState) error {
	state.IsEncounterActive = false
	return nil
}
func (e *EncounterEndedEvent) Message() string { return "Encounter Ended." }

// ActorAddedEvent brings a new entity into the encounter tracker.
type ActorAddedEvent struct {
	ID    string
	Name  string
	MaxHP int
}

func (e *ActorAddedEvent) Type() EventType { return EventActorAdded }
func (e *ActorAddedEvent) Apply(state *GameState) error {
	if _, ok := state.Entities[e.ID]; ok {
		return fmt.Errorf("actor with ID %s already tracking in encounter", e.ID)
	}

	state.Entities[e.ID] = &Entity{
		ID:    e.ID,
		Name:  e.Name,
		HP:    e.MaxHP,
		MaxHP: e.MaxHP,
	}
	state.TurnOrder = append(state.TurnOrder, e.ID)
	return nil
}
func (e *ActorAddedEvent) Message() string {
	return fmt.Sprintf("Added actor %s (%d/%d HP)", e.Name, e.MaxHP, e.MaxHP)
}

// TurnChangedEvent advances the current turn marker.
type TurnChangedEvent struct {
	ActorID string
}

func (e *TurnChangedEvent) Type() EventType { return EventTurnChanged }
func (e *TurnChangedEvent) Apply(state *GameState) error {
	// Simple lookup for turn index
	for i, id := range state.TurnOrder {
		if id == e.ActorID {
			state.CurrentTurn = i
			return nil
		}
	}
	return fmt.Errorf("actor %s not found in turn order", e.ActorID)
}
func (e *TurnChangedEvent) Message() string { return fmt.Sprintf("Turn changed to %s", e.ActorID) }

// HPChangedEvent modifies an actor's current HP (positive heals, negative damages).
type HPChangedEvent struct {
	ActorID string
	Amount  int
}

func (e *HPChangedEvent) Type() EventType { return EventHPChanged }
func (e *HPChangedEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("actor %s not found", e.ActorID)
	}

	ent.HP += e.Amount
	if ent.HP < 0 {
		ent.HP = 0
	}
	if ent.HP > ent.MaxHP {
		ent.HP = ent.MaxHP
	}
	return nil
}
func (e *HPChangedEvent) Message() string {
	if e.Amount > 0 {
		return fmt.Sprintf("%s healed for %d HP", e.ActorID, e.Amount)
	} else if e.Amount < 0 {
		return fmt.Sprintf("%s took %d damage", e.ActorID, -e.Amount)
	}
	return fmt.Sprintf("%s HP was unchanged", e.ActorID)
}

// DiceRolledEvent tracks a dice roll macro result.
type DiceRolledEvent struct {
	ActorName string
	Total     int
	RawRolls  []int
	Kept      []int
	Dropped   []int
	Modifier  int
}

func (e *DiceRolledEvent) Type() EventType { return EventDiceRolled }
func (e *DiceRolledEvent) Apply(state *GameState) error {
	return nil // Dice rolls do not inherently modify state
}
func (e *DiceRolledEvent) Message() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s rolled: %d\n", e.ActorName, e.Total))
	sb.WriteString(fmt.Sprintf("├─ Raw: %v\n", e.RawRolls))

	if len(e.Dropped) > 0 {
		sb.WriteString(fmt.Sprintf("├─ Kept: %v\n", e.Kept))
		sb.WriteString(fmt.Sprintf("├─ Dropped: %v\n", e.Dropped))
	}
	if e.Modifier != 0 {
		modPrefix := "+"
		if e.Modifier > 0 {
			modPrefix = ""
		}
		sb.WriteString(fmt.Sprintf("├─ Modifier: %s%d\n", modPrefix, e.Modifier))
	}
	return strings.TrimSpace(sb.String())
}

// InitiativeRolledEvent stores a participant's initiative roll and re-sorts turn order
type InitiativeRolledEvent struct {
	ActorName string
	Score     int
	IsManual  bool
	RawRolls  []int
	Kept      []int
	Dropped   []int
	Modifier  int
}

func (e *InitiativeRolledEvent) Type() EventType { return EventInitiativeRolled }
func (e *InitiativeRolledEvent) Apply(state *GameState) error {
	state.Initiatives[e.ActorName] = e.Score
	// In the future: Re-sort state.TurnOrder slice here based on state.Initiatives descending
	return nil
}
func (e *InitiativeRolledEvent) Message() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s rolled initiative: %d", e.ActorName, e.Score))

	if e.IsManual {
		sb.WriteString("\n├─ (Manual Override)")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("\n├─ Raw: %v", e.RawRolls))

	if len(e.Dropped) > 0 {
		sb.WriteString(fmt.Sprintf("\n├─ Kept: %v", e.Kept))
		sb.WriteString(fmt.Sprintf("\n├─ Dropped: %v", e.Dropped))
	}
	if e.Modifier != 0 {
		modPrefix := "+"
		if e.Modifier > 0 {
			modPrefix = ""
		}
		sb.WriteString(fmt.Sprintf("\n├─ Modifier: %s%d", modPrefix, e.Modifier))
	}
	return sb.String()
}
