package persistence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/suderio/ancient-draconic/internal/engine"
)

// EventWrapper facilitates serialization of polyphormic events
type EventWrapper struct {
	Type  engine.EventType `json:"type"`
	Event json.RawMessage  `json:"data"`
}

// Store handles append-only storing of event log.
type Store struct {
	file *os.File
}

// NewStore opens or creates the file at path for appending lines
func NewStore(path string) (*Store, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open store file: %w", err)
	}
	return &Store{file: file}, nil
}

// Append takes an Event interface and marshals it to jsonl log.
func (s *Store) Append(evt engine.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	wrapper := EventWrapper{
		Type:  evt.Type(),
		Event: data,
	}

	wrapperData, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}

	if _, err := s.file.Write(append(wrapperData, '\n')); err != nil {
		return err
	}
	return s.file.Sync()
}

// Load replays all jsonl strings and unpacks them to Event slice.
func (s *Store) Load() ([]engine.Event, error) {
	var events []engine.Event

	// Reset file pointer to beginning
	if _, err := s.file.Seek(0, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(s.file)
	for scanner.Scan() {
		var wrapper EventWrapper
		if err := json.Unmarshal(scanner.Bytes(), &wrapper); err != nil {
			return nil, fmt.Errorf("failed to decode wrapper: %w", err)
		}

		var evt engine.Event
		switch wrapper.Type {
		case engine.EventEncounterStarted:
			evt = &engine.EncounterStartedEvent{}
		case engine.EventActorAdded:
			evt = &engine.ActorAddedEvent{}
		case engine.EventTurnChanged:
			evt = &engine.TurnChangedEvent{}
		case engine.EventHPChanged:
			evt = &engine.HPChangedEvent{}
		case engine.EventDiceRolled:
			evt = &engine.DiceRolledEvent{}
		case engine.EventEncounterEnded:
			evt = &engine.EncounterEndedEvent{}
		case engine.EventInitiativeRolled:
			evt = &engine.InitiativeRolledEvent{}
		case engine.EventAttackResolved:
			evt = &engine.AttackResolvedEvent{}
		case engine.EventTurnEnded:
			evt = &engine.TurnEndedEvent{}
		case engine.EventAskIssued:
			evt = &engine.AskIssuedEvent{}
		case engine.EventCheckResolved:
			evt = &engine.CheckResolvedEvent{}
		case engine.EventConditionApplied:
			evt = &engine.ConditionAppliedEvent{}
		default:
			return nil, fmt.Errorf("unknown event type in log: %s", wrapper.Type)
		}

		if err := json.Unmarshal(wrapper.Event, evt); err != nil {
			return nil, fmt.Errorf("failed to parse event data into specific type: %w", err)
		}

		events = append(events, evt)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// Close handles safe shutdown.
func (s *Store) Close() error {
	return s.file.Close()
}
