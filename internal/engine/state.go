package engine

// GameState is the actively calculated projection of the game session.
type GameState struct {
	IsEncounterActive bool                `json:"is_encounter_active"`
	Initiatives       map[string]int      `json:"initiatives"`
	Entities          map[string]*Entity  `json:"entities"`
	TurnOrder         []string            `json:"turn_order"`
	CurrentTurn       int                 `json:"current_turn"`
	PendingDamage     *PendingDamageState `json:"pending_damage"`
}

// PendingDamageState tracks weapon hits for the next sequential damage command
type PendingDamageState struct {
	Attacker  string          `json:"attacker"`
	Targets   []string        `json:"targets"`
	Weapon    string          `json:"weapon"`
	HitStatus map[string]bool `json:"hit_status"`
}

// Entity represents an actor (Monster, Player, NPC) participating in the session
type Entity struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"` // "Character" or "Monster"
	Name       string   `json:"name"`
	HP         int      `json:"hp"`
	MaxHP      int      `json:"max_hp"`
	Conditions []string `json:"conditions"`
}

// NewGameState creates an empty clean slate
func NewGameState() *GameState {
	return &GameState{
		Entities:    make(map[string]*Entity),
		TurnOrder:   make([]string, 0),
		Initiatives: make(map[string]int),
	}
}

// IsFrozen checks if the active encounter is blocked by missing initiative rolls
func (s *GameState) IsFrozen() bool {
	if !s.IsEncounterActive {
		return false
	}
	for id := range s.Entities {
		if _, ok := s.Initiatives[id]; !ok {
			return true
		}
	}
	return false
}
