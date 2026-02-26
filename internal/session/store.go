package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/suderio/ancient-draconic/internal/engine"
)

// EventWrapper serializes polymorphic engine events to JSONL.
type EventWrapper struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Store handles append-only storage of engine events as JSONL.
type Store struct {
	file *os.File
}

// NewStore opens or creates a JSONL event log at the given path.
func NewStore(path string) (*Store, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open event store: %w", err)
	}
	return &Store{file: file}, nil
}

// Append marshals an engine Event and appends it as a JSONL line.
func (s *Store) Append(evt engine.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	wrapper := EventWrapper{
		Type: evt.Type(),
		Data: data,
	}

	line, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal wrapper: %w", err)
	}

	if _, err := s.file.Write(append(line, '\n')); err != nil {
		return err
	}
	return s.file.Sync()
}

// Load replays all events from the JSONL log and returns them.
func (s *Store) Load() ([]engine.Event, error) {
	if _, err := s.file.Seek(0, 0); err != nil {
		return nil, err
	}

	var events []engine.Event
	scanner := bufio.NewScanner(s.file)
	for scanner.Scan() {
		var wrapper EventWrapper
		if err := json.Unmarshal(scanner.Bytes(), &wrapper); err != nil {
			return nil, fmt.Errorf("failed to decode event wrapper: %w", err)
		}

		evt, err := unmarshalEvent(wrapper.Type, wrapper.Data)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, scanner.Err()
}

// Close flushes and closes the underlying file.
func (s *Store) Close() error {
	return s.file.Close()
}

// unmarshalEvent reconstructs a concrete Event from its type discriminator and JSON data.
func unmarshalEvent(typeName string, data json.RawMessage) (engine.Event, error) {
	var evt engine.Event

	switch typeName {
	case "LoopEvent":
		evt = &engine.LoopEvent{}
	case "LoopOrderAscendingEvent":
		evt = &engine.LoopOrderAscendingEvent{}
	case "LoopOrderEvent":
		evt = &engine.LoopOrderEvent{}
	case "ActorAddedEvent":
		evt = &engine.ActorAddedEvent{}
	case "AttributeChangedEvent":
		evt = &engine.AttributeChangedEvent{}
	case "AddSpentEvent":
		evt = &engine.AddSpentEvent{}
	case "ConditionEvent":
		evt = &engine.ConditionEvent{}
	case "AskIssuedEvent":
		evt = &engine.AskIssuedEvent{}
	case "HintEvent":
		evt = &engine.HintEvent{}
	case "DiceRolledEvent":
		evt = &engine.DiceRolledEvent{}
	case "MetadataChangedEvent":
		evt = &engine.MetadataChangedEvent{}
	case "CheckEvent":
		evt = &engine.CheckEvent{}
	default:
		return nil, fmt.Errorf("unknown event type: %s", typeName)
	}

	if err := json.Unmarshal(data, evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", typeName, err)
	}
	return evt, nil
}
