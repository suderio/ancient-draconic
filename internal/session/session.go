package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/suderio/ancient-draconic/internal/engine"
)

// Session manages the game loop: parsing input, executing commands,
// persisting events, and maintaining game state.
type Session struct {
	mu       sync.Mutex
	manifest *engine.Manifest
	state    *engine.GameState
	store    *Store
	eval     *engine.LuaEvaluator
	dataDirs []string
}

// NewSession bootstraps a manifest-driven game session.
func NewSession(dataDirs []string, storePath string) (*Session, error) {
	// 1. Create Lua evaluator
	eval, err := engine.NewLuaEvaluator(nil) // Use default dice roller
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	// 2. Load manifest from the first available location
	m, err := findAndLoadManifest(dataDirs, eval)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
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
// It is thread-safe for concurrent calls (e.g., from TUI and Telegram).
func (s *Session) Execute(input string) ([]engine.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	var finalEvents []engine.Event

	for _, evt := range events {
		if req, ok := evt.(*engine.UndoRequestEvent); ok {
			return s.handleUndoRequest(req)
		}
		if err := s.applyAndPersist(evt); err != nil {
			return nil, err
		}
		finalEvents = append(finalEvents, evt)
	}

	return finalEvents, nil
}

// handleUndoRequest delegates the engine's undo request to session log rewinding logic.
func (s *Session) handleUndoRequest(req *engine.UndoRequestEvent) ([]engine.Event, error) {
	if req.Turn > 0 {
		return s.undoToBoundary("TurnStartedEvent", req.Turn)
	}
	if req.Round > 0 {
		return s.undoToBoundary("RoundStartedEvent", req.Round)
	}

	steps := req.Steps
	if steps < 1 {
		return nil, fmt.Errorf("steps must be at least 1")
	}

	undone, err := s.Undo(steps)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Undid %d event(s). State rewound.", undone)
	return []engine.Event{&engine.HintEvent{MessageStr: msg}}, nil
}

// undoToBoundary walks the event log backwards to find the Nth occurrence of the
// given boundary event type and truncates the log to that point.
func (s *Session) undoToBoundary(eventType string, count int) ([]engine.Event, error) {
	events, err := s.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if count < 1 {
		count = 1
	}

	found := 0
	keepN := len(events)
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type() == eventType {
			found++
			if found >= count {
				keepN = i
				break
			}
		}
	}

	if found < count {
		return nil, fmt.Errorf("cannot undo %d %s(s): only %d found in the log", count, eventType, found)
	}

	undone := len(events) - keepN
	if undone == 0 {
		return []engine.Event{&engine.HintEvent{MessageStr: "Nothing to undo."}}, nil
	}

	if _, err := s.Undo(undone); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Undid %d event(s) to %s boundary. State rewound.", undone, eventType)
	return []engine.Event{&engine.HintEvent{MessageStr: msg}}, nil
}

// parseIntParam extracts an int from various types, returning defaultVal on failure.
func parseIntParam(v any, defaultVal int) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case string:
		var i int
		if _, err := fmt.Sscanf(n, "%d", &i); err == nil {
			return i
		}
	}
	return defaultVal
}

// isGM checks if the actor is the Game Master.
func isGM(actorID string) bool {
	return strings.ToUpper(actorID) == "GM"
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

// Undo removes the last `steps` events from the log and rebuilds state from scratch.
// This is a GM-only operation enforced by the hardcoded command dispatcher.
func (s *Session) Undo(steps int) (int, error) {
	total, err := s.store.EventCount()
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	if steps > total {
		return 0, fmt.Errorf("cannot undo %d events: only %d events in the log", steps, total)
	}

	keepN := total - steps
	if err := s.store.Truncate(keepN); err != nil {
		return 0, fmt.Errorf("failed to truncate event log: %w", err)
	}

	// Rebuild state from the truncated log
	s.state = engine.NewGameState()
	if err := s.rebuildState(); err != nil {
		return 0, fmt.Errorf("failed to rebuild state after undo: %w", err)
	}
	if err := s.loadEntities(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	return steps, nil
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

// findAndLoadManifest searches data directories for a manifest.lua or manifest.yaml file.
func findAndLoadManifest(dataDirs []string, eval *engine.LuaEvaluator) (*engine.Manifest, error) {
	for _, dir := range dataDirs {
		luaPath := filepath.Join(dir, "manifest.lua")
		if _, err := os.Stat(luaPath); err == nil {
			return eval.LoadManifestLua(luaPath)
		}

		yamlPath := filepath.Join(dir, "manifest.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			return engine.LoadManifest(yamlPath) // legacy fallback
		}
	}
	return nil, fmt.Errorf("neither manifest.lua nor manifest.yaml found in any of: %s", strings.Join(dataDirs, ", "))
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
