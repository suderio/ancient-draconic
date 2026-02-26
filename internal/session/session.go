package session

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suderio/ancient-draconic/internal/engine"
)

// Session manages the game loop: parsing input, executing commands,
// persisting events, and maintaining game state.
type Session struct {
	manifest *engine.Manifest
	state    *engine.GameState
	store    *Store
	eval     *engine.Evaluator
	dataDirs []string
}

// NewSession bootstraps a manifest-driven game session.
func NewSession(dataDirs []string, storePath string) (*Session, error) {
	// 1. Load manifest from the first available location
	m, err := findAndLoadManifest(dataDirs)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// 2. Create CEL evaluator
	eval, err := engine.NewEvaluator(nil) // Use default dice roller
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	// 3. Open event store
	store, err := NewStore(storePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open event store: %w", err)
	}

	s := &Session{
		manifest: m,
		state:    engine.NewGameState(),
		store:    store,
		eval:     eval,
		dataDirs: dataDirs,
	}

	// 4. Rebuild state from event log
	if err := s.rebuildState(); err != nil {
		store.Close()
		return nil, err
	}

	// 5. Load entity data files (characters/monsters) into state
	if err := s.loadEntities(); err != nil {
		// Non-fatal: entities can be added via commands too
		fmt.Printf("Warning: %v\n", err)
	}

	return s, nil
}

// Execute takes a raw command string, parses it, executes the command,
// applies and persists the resulting events, and returns them.
func (s *Session) Execute(input string) ([]engine.Event, error) {
	parsed := ParseInput(input)

	if parsed.Command == "" {
		return nil, fmt.Errorf("empty command")
	}

	events, err := engine.ExecuteCommand(
		parsed.Command,
		parsed.ActorID,
		parsed.Targets,
		parsed.Params,
		s.state,
		s.manifest,
		s.eval,
	)
	if err != nil {
		return nil, err
	}

	for _, evt := range events {
		if err := s.applyAndPersist(evt); err != nil {
			return nil, err
		}
	}

	return events, nil
}

// State returns the current game state.
func (s *Session) State() *engine.GameState {
	return s.state
}

// Manifest returns the loaded manifest for autocomplete and help.
func (s *Session) Manifest() *engine.Manifest {
	return s.manifest
}

// Close releases resources held by the session.
func (s *Session) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// rebuildState replays all persisted events to reconstruct the in-memory state.
func (s *Session) rebuildState() error {
	events, err := s.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load event log: %w", err)
	}

	for _, evt := range events {
		if err := evt.Apply(s.state); err != nil {
			return fmt.Errorf("failed to replay event %s: %w", evt.Type(), err)
		}
	}

	return nil
}

// applyAndPersist commits an event to both the in-memory state and the persistent store.
func (s *Session) applyAndPersist(evt engine.Event) error {
	// HintEvents are display-only and should not be persisted
	if _, isHint := evt.(*engine.HintEvent); isHint {
		return nil
	}

	if err := s.store.Append(evt); err != nil {
		return fmt.Errorf("failed to persist event: %w", err)
	}

	if err := evt.Apply(s.state); err != nil {
		return fmt.Errorf("failed to apply event: %w", err)
	}

	return nil
}

// findAndLoadManifest searches data directories for a manifest.yaml file.
func findAndLoadManifest(dataDirs []string) (*engine.Manifest, error) {
	for _, dir := range dataDirs {
		path := filepath.Join(dir, "manifest.yaml")
		m, err := engine.LoadManifest(path)
		if err == nil {
			return m, nil
		}
	}
	return nil, fmt.Errorf("manifest.yaml not found in any of: %s", strings.Join(dataDirs, ", "))
}

// loadEntities scans data directories for character and monster YAML files.
func (s *Session) loadEntities() error {
	for _, dir := range s.dataDirs {
		for _, sub := range []string{"characters", "monsters", "data/characters", "data/monsters"} {
			entities, _ := loadEntitiesFromDir(filepath.Join(dir, sub))
			for _, e := range entities {
				if _, exists := s.state.Entities[e.ID]; !exists {
					s.state.Entities[e.ID] = e
				}
			}
		}
	}
	return nil
}

// loadEntitiesFromDir reads all YAML files in a directory and loads them as entities.
func loadEntitiesFromDir(dir string) ([]*engine.Entity, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil || len(entries) == 0 {
		return nil, err
	}

	var result []*engine.Entity
	for _, path := range entries {
		e, err := engine.LoadEntity(path)
		if err != nil {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}
