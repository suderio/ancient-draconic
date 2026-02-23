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
		events, err := command.ExecuteEncounter(astCmd.Encounter, s.state, s.loader, s.registry)
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
		events, err := command.ExecuteAdd(astCmd.Add, s.state, s.loader, s.registry)
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
		actorID := "GM"
		if astCmd.Initiative.Actor != nil {
			actorID = astCmd.Initiative.Actor.Name
		}
		events, err := command.ExecuteGenericCommand("initiative", actorID, []string{actorID}, nil, input, s.state, s.loader, s.registry)
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
	} else if astCmd.Attack != nil {
		actorID, err := command.ResolveActor(astCmd.Attack.Actor, s.state)
		if err != nil {
			if err == engine.ErrSilentIgnore {
				return nil, nil
			}
			return nil, err
		}

		params := map[string]any{
			"weapon":      astCmd.Attack.Weapon,
			"offhand":     astCmd.Attack.OffHand,
			"opportunity": astCmd.Attack.Opportunity,
		}
		events, err := command.ExecuteGenericCommand("attack", actorID, astCmd.Attack.Targets, params, input, s.state, s.loader, s.registry)
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
	} else if astCmd.Damage != nil {
		if s.state.PendingDamage == nil || s.state.IsFrozen() {
			return nil, nil // Silently ignore as per legacy
		}

		targets := []string{}
		for _, t := range s.state.PendingDamage.Targets {
			if s.state.PendingDamage.HitStatus[t] {
				targets = append(targets, t)
			}
		}

		if len(targets) == 0 {
			return nil, nil
		}

		params := map[string]any{
			"weapon":  s.state.PendingDamage.Weapon,
			"offhand": s.state.PendingDamage.IsOffHand,
		}
		// In a real scenario, we'd lookup the actual weapon dice/type from loader here
		// but for the transition, we'll let ExecuteGenericCommand do its best or pass them in.
		// Actually, ExecuteGenericCommand already calls ResolveEntityAction.

		events, err := command.ExecuteGenericCommand("damage", s.state.PendingDamage.Attacker, targets, params, input, s.state, s.loader, s.registry)
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
	} else if astCmd.Turn != nil {
		actorID := "GM"
		if astCmd.Turn.Actor != nil {
			actorID = astCmd.Turn.Actor.Name
		}
		events, err := command.ExecuteGenericCommand("turn", actorID, []string{actorID}, nil, input, s.state, s.loader, s.registry)
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
		actorID := "GM"
		if astCmd.Dodge.Actor != nil {
			actorID = astCmd.Dodge.Actor.Name
		}
		events, err := command.ExecuteGenericCommand("dodge", actorID, []string{actorID}, nil, input, s.state, s.loader, s.registry)
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
	} else if astCmd.Grapple != nil {
		actorID := "GM"
		if astCmd.Grapple.Actor != nil {
			actorID = astCmd.Grapple.Actor.Name
		}
		params := map[string]any{}
		events, err := command.ExecuteGenericCommand("grapple", actorID, []string{astCmd.Grapple.Target}, params, input, s.state, s.loader, s.registry)
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
			actorID := "GM"
			if astCmd.Action.Actor != nil {
				actorID = astCmd.Action.Actor.Name
			}
			params := map[string]any{}
			events, err = command.ExecuteGenericCommand("shove", actorID, []string{astCmd.Action.Target}, params, input, s.state, s.loader, s.registry)
		} else {
			events, err = command.ExecuteAction(astCmd.Action, s.state, s.loader, s.registry)
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
		events, err := command.ExecuteHelpAction(astCmd.HelpAction, s.state, s.registry)
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
		actorID, err := command.ResolveActor(astCmd.Check.Actor, s.state)
		if err != nil {
			if err == engine.ErrSilentIgnore {
				return nil, nil
			}
			return nil, err
		}

		params := map[string]any{
			"check": strings.Join(astCmd.Check.Check, " "),
		}
		events, err := command.ExecuteGenericCommand("check", actorID, []string{actorID}, params, input, s.state, s.loader, s.registry)
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
