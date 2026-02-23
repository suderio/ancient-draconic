package engine

import (
	"github.com/suderio/ancient-draconic/internal/data"
)

// GameState is the actively calculated projection of the game session.
type GameState struct {
	IsEncounterActive   bool                          `json:"is_encounter_active"`
	Initiatives         map[string]int                `json:"initiatives"`
	Entities            map[string]*Entity            `json:"entities"`
	TurnOrder           []string                      `json:"turn_order"`
	CurrentTurn         int                           `json:"current_turn"`
	PendingDamage       *PendingDamageState           `json:"pending_damage"`
	PendingChecks       map[string]*PendingCheckState `json:"pending_checks"`
	PendingAdjudication *PendingAdjudicationState     `json:"pending_adjudication"`
	SpentRecharges      map[string][]string           `json:"spent_recharges"`
}

// RollConsequence tracks the automated impacts of parsing an Ask string
type RollConsequence struct {
	IsDamage        bool   `json:"is_damage"`
	DamageDice      string `json:"damage_dice"`
	HalfDamage      bool   `json:"half_damage"`
	Condition       string `json:"condition"`
	RemoveCondition string `json:"remove_condition"`
}

// PendingCheckState tracks a required roll requested by the GM
type PendingCheckState struct {
	Check    []string         `json:"check"`
	DC       int              `json:"dc"`
	Fails    *RollConsequence `json:"fails"`
	Succeeds *RollConsequence `json:"succeeds"`
}

// PendingDamageState tracks weapon hits for the next sequential damage command
type PendingDamageState struct {
	Attacker  string          `json:"attacker"`
	Targets   []string        `json:"targets"`
	Weapon    string          `json:"weapon"`
	HitStatus map[string]bool `json:"hit_status"`
	IsOffHand bool            `json:"is_off_hand"`
}

// PendingAdjudicationState tracks a command waiting for GM approval
type PendingAdjudicationState struct {
	OriginalCommand string `json:"original_command"`
	Approved        bool   `json:"approved"`
}

// Entity represents an actor (Monster, Player, NPC) participating in the session
type Entity struct {
	ID         string         `json:"id"`
	Category   string         `json:"category"`    // "Character" or "Monster"
	EntityType string         `json:"entity_type"` // Genre-specific type: "undead", "humanoid", etc.
	Name       string         `json:"name"`
	HP         int            `json:"hp"`
	MaxHP      int            `json:"max_hp"`
	Conditions []string       `json:"conditions"`
	Stats      map[string]int `json:"stats"`     // Generic stats: str, dex, technical, etc.
	Resources  map[string]int `json:"resources"` // Tracked integers: spell_slots, luck, etc.
	Abilities  []data.Ability `json:"abilities"`

	ActionsRemaining      int `json:"actions_remaining"`
	BonusActionsRemaining int `json:"bonus_actions_remaining"`
	ReactionsRemaining    int `json:"reactions_remaining"`
	AttacksRemaining      int `json:"attacks_remaining"`

	HasAttackedThisTurn    bool   `json:"has_attacked_this_turn"`
	LastAttackedWithWeapon string `json:"last_attacked_with_weapon"`
}

// NewGameState creates an empty clean slate
func NewGameState() *GameState {
	return &GameState{
		Entities:       make(map[string]*Entity),
		TurnOrder:      make([]string, 0),
		Initiatives:    make(map[string]int),
		PendingChecks:  make(map[string]*PendingCheckState),
		SpentRecharges: make(map[string][]string),
		CurrentTurn:    -1,
	}
}

// IsFrozen checks if the active encounter is blocked by missing initiative rolls or GM-requested checks
func (s *GameState) IsFrozen() bool {
	if len(s.PendingChecks) > 0 {
		return true
	}
	if s.PendingAdjudication != nil && !s.PendingAdjudication.Approved {
		return true
	}
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
