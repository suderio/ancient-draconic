package session

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/suderio/dndsl/internal/command"
	"github.com/suderio/dndsl/internal/data"
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

// Store defines the dependency required by Session to persist events
type Store interface {
	Append(evt engine.Event) error
	Load() ([]engine.Event, error)
	Close() error
}

// Session manages the cohesive loop of taking commands, executing them, persisting events, and projecting GameState
type Session struct {
	loader *data.Loader
	store  Store
	state  *engine.GameState
}

// NewSession bootstraps a game session pipeline relying on an injected store
func NewSession(dataDirs []string, store Store) (*Session, error) {
	s := &Session{loader: data.NewLoader(dataDirs), store: store}
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
		events, err := command.ExecuteEncounter(astCmd.Encounter, s.state, s.loader)
		if err != nil {
			return nil, err
		}
		for _, evt := range events {
			if err := s.ApplyAndAppend(evt); err != nil {
				return nil, err
			}
		}
		// Return the defining top-level event as the printable anchor
		return events[0], nil
	}

	if astCmd.Add != nil {
		events, err := command.ExecuteAdd(astCmd.Add, s.state, s.loader)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	}

	if astCmd.Initiative != nil {
		events, err := command.ExecuteInitiative(astCmd.Initiative, s.state, s.loader)
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
		return events[0], nil
	} else if astCmd.Attack != nil {
		events, err := command.ExecuteAttack(astCmd.Attack, s.state, s.loader)
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
		return events[0], nil
	} else if astCmd.Damage != nil {
		events, err := command.ExecuteDamage(astCmd.Damage, s.state, s.loader)
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
		return events[0], nil
	} else if astCmd.Turn != nil {
		events, err := command.ExecuteTurn(astCmd.Turn, s.state, s.loader)
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
		return events[0], nil
	} else if astCmd.Hint != nil {
		events, err := command.ExecuteHint(astCmd.Hint, s.state)
		if err != nil {
			return nil, err
		}
		// Hints are stateless queries; we do not append them to the log
		return events[0], nil
	} else if astCmd.Adjudicate != nil {
		events, err := command.ExecuteAdjudicate(astCmd.Adjudicate)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	} else if astCmd.Allow != nil {
		originalStr := ""
		if s.state.PendingAdjudication != nil {
			originalStr = s.state.PendingAdjudication.OriginalCommand
		}
		events, err := command.ExecuteAllow(astCmd.Allow, s.state)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		if originalStr != "" {
			return s.Execute(originalStr)
		}
		return events[0], nil
	} else if astCmd.Deny != nil {
		events, err := command.ExecuteDeny(astCmd.Deny, s.state)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	} else if astCmd.Dodge != nil {
		events, err := command.ExecuteDodge(astCmd.Dodge, s.state)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	} else if astCmd.Grapple != nil {
		events, err := command.ExecuteGrapple(astCmd.Grapple, s.state, s.loader)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	} else if astCmd.Action != nil {
		var events []engine.Event
		var err error
		if strings.ToLower(astCmd.Action.Action) == "shove" {
			events, err = command.ExecuteShove(astCmd.Action, s.state, s.loader)
		} else {
			events, err = command.ExecuteAction(astCmd.Action, s.state, s.loader)
		}
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	} else if astCmd.HelpAction != nil {
		events, err := command.ExecuteHelpAction(astCmd.HelpAction, s.state)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if err := s.ApplyAndAppend(e); err != nil {
				return nil, err
			}
		}
		return events[0], nil
	} else if astCmd.Ask != nil {
		events, err := command.ExecuteAsk(astCmd.Ask, s.state)
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
		return events[0], nil
	} else if astCmd.Check != nil {
		events, err := command.ExecuteCheck(astCmd.Check, s.state, s.loader)
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
		return events[0], nil
	} else if astCmd.Help != nil {
		events, err := command.ExecuteHelp(astCmd.Help, s.state)
		if err != nil {
			return nil, err
		}
		return events[0], nil
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
