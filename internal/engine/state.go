package engine

// GameState is the actively calculated projection of the game session.
type GameState struct {
	IsEncounterActive bool               `json:"is_encounter_active"`
	Entities          map[string]*Entity `json:"entities"`
	TurnOrder         []string           `json:"turn_order"`
	CurrentTurn       int                `json:"current_turn"`

	// System-agnostic state tracking (e.g., "pending_damage", "initiatives")
	Metadata map[string]any `json:"metadata"`

	// Manifest-driven freeze logic: contain a CEL expression that must be false to unfreeze
	FrozenUntil string `json:"frozen_until"`
}

// Entity represents an actor participating in the session with a generic data model.
type Entity struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Types           []string          `json:"types"`         // e.g., "monster", "undead", "humanoid"
	Classes         map[string]string `json:"classes"`       // e.g., "size": "medium", "class": "fighter"
	Stats           map[string]int    `json:"stats"`         // e.g., "str": 16, "dex": 14
	Resources       map[string]int    `json:"resources"`     // Max values for tracked integers (e.g., "hp": 20)
	Spent           map[string]int    `json:"spent"`         // Current usage of resources (e.g., "hp": 5 means 15 current HP)
	Conditions      []string          `json:"conditions"`    // Temporary conditions (e.g., "poisoned")
	Proficiencies   map[string]int    `json:"proficiencies"` // e.g., "athletics": 2, "saving-throw-dex": 2
	Statuses        map[string]string `json:"statuses"`      // Arbitrary state (e.g., "concentrating": "true")
	Inventory       map[string]int    `json:"inventory"`     // Items and counts
	Immunities      []string          `json:"immunities"`
	Resistances     []string          `json:"resistances"`
	Vulnerabilities []string          `json:"vulnerabilities"`
}

// NewGameState creates an empty clean slate
func NewGameState() *GameState {
	return &GameState{
		Entities:    make(map[string]*Entity),
		TurnOrder:   make([]string, 0),
		Metadata:    make(map[string]any),
		CurrentTurn: -1,
	}
}

// IsFrozen checks if the active encounter is blocked by manifest-driven requirements
func (s *GameState) IsFrozen() bool {
	// 1. Check explicit manifest freeze (e.g. "is_frozen: true" or "dc_check_pending: true")
	if s.FrozenUntil != "" {
		return true
	}

	// 2. Check transient metadata that implies freeze (legacy compatibility/helper)
	if pendingChecks, ok := s.Metadata["pending_checks"].(map[string]any); ok && len(pendingChecks) > 0 {
		return true
	}
	if pendingAdj, ok := s.Metadata["pending_adjudication"].(map[string]any); ok {
		if approved, ok := pendingAdj["approved"].(bool); ok && !approved {
			return true
		}
	}

	if !s.IsEncounterActive {
		return false
	}

	// 3. System-level freeze (e.g. missing initiative)
	initiatives, ok := s.Metadata["initiatives"].(map[string]int)
	if !ok {
		// If no initiative map at all, and encounter active, we might be frozen if there are entities
		return len(s.Entities) > 0
	}

	for id := range s.Entities {
		if _, ok := initiatives[id]; !ok {
			return true
		}
	}
	return false
}
