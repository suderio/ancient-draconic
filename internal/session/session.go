package session

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/suderio/ancient-draconic/internal/command"
	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

// Store defines the dependency required by Session to persist events
type Store interface {
	Append(evt engine.Event) error
	Load() ([]engine.Event, error)
	Close() error
}

// Session manages the cohesive loop of taking commands, executing them, persisting events, and projecting GameState
type Session struct {
	loader   *data.Loader
	store    Store
	state    *engine.GameState
	registry *rules.Registry
}

// NewSession bootstraps a game session pipeline relying on an injected store
func NewSession(dataDirs []string, store Store) (*Session, error) {
	loader := data.NewLoader(dataDirs)
	manifest, err := loader.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load campaign manifest: %w", err)
	}

	// Bridge rules.Registry to engine.Roll
	reg, err := rules.NewRegistry(manifest, func(s string) int {
		expr := &parser.DiceExpr{Raw: s}
		res, _ := engine.Roll(expr)
		return res.Total
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rules registry: %w", err)
	}

	s := &Session{
		loader:   loader,
		store:    store,
		registry: reg,
	}
	if err := s.RebuildState(); err != nil {
		return nil, err
	}
	return s, nil
}

// RebuildState reads the entire event log from the store and projects the latest GameState
func (s *Session) RebuildState() error {
	events, err := s.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load event log: %w", err)
	}

	proj := engine.NewProjector()
	state, err := proj.Build(events)
	if err != nil {
		return fmt.Errorf("failed to project game state: %w", err)
	}

	s.state = state
	return nil
}

// State returns the current projected GameState
func (s *Session) State() *engine.GameState {
	return s.state
}

// Loader returns the instantiated YAML reference engine
func (s *Session) Loader() *data.Loader {
	return s.loader
}

// Execute takes a raw command string from a UI client, coordinates execution, appends the result, and returns the descriptive Event
func (s *Session) Execute(input string) (engine.Event, error) {
	langParser := parser.Build()

	// Let's intercept legacy fake commands temporarily here before we properly build ASTs for them
	parts := strings.Split(input, " ")
	if parts[0] == "heal" {
		return s.executeLegacyPseudoCommand(parts)
	}

	astCmd, err := langParser.ParseString("", input)
	if err != nil {
		return nil, parser.MapError(input, err)
	}

	if astCmd.Roll != nil {
		evt, err := command.ExecuteRoll(astCmd.Roll)
		if err != nil {
			return nil, fmt.Errorf("roll execution error: %w", err)
		}

		if err := s.ApplyAndAppend(evt); err != nil {
			return nil, err
		}

		return evt, nil
	}

	if astCmd.Encounter != nil {
		actorID := "GM"
		if astCmd.Encounter.Actor != nil {
			actorID = astCmd.Encounter.Actor.Name
		}
		params := map[string]any{
			"action": astCmd.Encounter.Action,
		}
		events, err := command.ExecuteGenericCommand("encounter", actorID, astCmd.Encounter.Targets, params, input, s.state, s.loader, s.registry)
		if err != nil {
			return nil, err
		}
		for _, evt := range events {
			if err := s.ApplyAndAppend(evt); err != nil {
				return nil, err
			}
		}
		if len(events) > 0 {
			return events[0], nil
		}
		return nil, nil
	}

	if astCmd.Add != nil {
		actorID := "GM"
		if astCmd.Add.Actor != nil {
			actorID = astCmd.Add.Actor.Name
		}
		events, err := command.ExecuteGenericCommand("add", actorID, astCmd.Add.Targets, nil, input, s.state, s.loader, s.registry)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		if len(events) > 0 {
			return events[0], nil
		}
		return nil, nil
	}

	if astCmd.Generic != nil {
		actorID := "GM"
		if astCmd.Generic.Actor != nil {
			var err error
			actorID, err = command.ResolveActor(astCmd.Generic.Actor, s.state)
			if err != nil {
				if err == engine.ErrSilentIgnore {
					return nil, nil
				}
				return nil, err
			}
		}

		params := map[string]any{}
		var targets []string

		for _, arg := range astCmd.Generic.Args {
			key := strings.TrimSuffix(arg.Key, ":")
			if key == "to" || key == "of" {
				targets = arg.Values
			} else if len(arg.Values) == 1 {
				// Single value, try parsing as bool if it's "true", else string/int mapping happens inside
				params[key] = arg.Values[0]
			} else if len(arg.Values) > 1 {
				params[key] = arg.Values
			}
		}

		// Turn has very specific game loop progression rules
		if strings.ToLower(astCmd.Generic.Name) == "turn" {
			// 1. End turn for current actor
			if len(s.state.TurnOrder) > 0 && s.state.CurrentTurn >= 0 {
				currentActor := s.state.TurnOrder[s.state.CurrentTurn]
				if endEvents, err := command.ExecuteGenericCommand("end_turn", currentActor, []string{currentActor}, nil, "end_turn", s.state, s.loader, s.registry); err == nil {
					for _, e := range endEvents {
						s.ApplyAndAppend(e)
					}
				}
				s.processExpirations("end_turn", currentActor)
			}

			// 2. Advance turn
			events, err := command.ExecuteGenericCommand("turn", actorID, []string{actorID}, nil, input, s.state, s.loader, s.registry)
			if err != nil {
				return nil, err
			}
			var firstEvent engine.Event
			for _, e := range events {
				if err := s.ApplyAndAppend(e); err != nil {
					return nil, err
				}
				if firstEvent == nil {
					firstEvent = e
				}
			}

			// 3. Start turn for next actor
			if len(s.state.TurnOrder) > 0 && s.state.CurrentTurn >= 0 {
				newActor := s.state.TurnOrder[s.state.CurrentTurn]
				if startEvents, err := command.ExecuteGenericCommand("start_turn", newActor, []string{newActor}, nil, "start_turn", s.state, s.loader, s.registry); err == nil {
					for _, e := range startEvents {
						s.ApplyAndAppend(e)
					}
				}
				s.processExpirations("start_turn", newActor)
			}

			if firstEvent != nil {
				return firstEvent, nil
			}
			return nil, nil
		}

		events, err := command.ExecuteGenericCommand(strings.ToLower(astCmd.Generic.Name), actorID, targets, params, input, s.state, s.loader, s.registry)
		if err != nil {
			if err == engine.ErrSilentIgnore {
				return nil, nil
			}
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		if len(events) > 0 {
			return events[0], nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("unsupported command pattern")
}

// ApplyAndAppend commits a finalized event to the store and updates memory
func (s *Session) ApplyAndAppend(evt engine.Event) error {
	if err := s.store.Append(evt); err != nil {
		return fmt.Errorf("failed to persist event log: %w", err)
	}

	if err := evt.Apply(s.state); err != nil {
		// Log corruption warning, but in production we might trigger a full rebuild instead
		return fmt.Errorf("failed to apply event to memory state: %w", err)
	}

	return nil
}

func (s *Session) executeLegacyPseudoCommand(parts []string) (engine.Event, error) {
	// ... Quick mapping for add/damage/heal to maintain REPL backwards compatibility
	// until we write true Participle AST structures for them later

	var evt engine.Event

	switch parts[0] {
	case "heal":
		if len(parts) == 3 {
			amt, _ := strconv.Atoi(parts[2])
			evt = &engine.HPChangedEvent{
				ActorID: parts[1],
				Amount:  amt,
			}
		} else {
			return nil, fmt.Errorf("Usage: heal <id> <amount>")
		}
	}

	if evt != nil {
		if err := s.ApplyAndAppend(evt); err != nil {
			return nil, err
		}
		return evt, nil
	}

	return nil, fmt.Errorf("unrecognized legacy command")
}
func (s *Session) processExpirations(trigger string, currentActor string) error {
	var events []engine.Event
	if expMap, ok := s.state.Metadata["conditions_expiry"].(map[string]any); ok {
		// Because maps might be modified or iterated, we collect first
		var keysToDelete []string
		for key, val := range expMap {
			if vMap, ok := val.(map[string]string); ok {
				if vMap["expires_on"] == trigger && vMap["reference_actor"] == currentActor {
					parts := strings.SplitN(key, ":", 2)
					if len(parts) == 2 {
						events = append(events, &engine.ConditionRemovedEvent{
							ActorID:   parts[0],
							Condition: parts[1],
						})
						keysToDelete = append(keysToDelete, key)
					}
				}
			}
		}
		// Clean up the tracking state directly so it's not repeatedly checked
		for _, k := range keysToDelete {
			delete(expMap, k)
		}
	}
	for _, e := range events {
		if err := s.ApplyAndAppend(e); err != nil {
			return err
		}
	}
	return nil
}
