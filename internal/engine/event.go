package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/suderio/ancient-draconic/internal/data"
)

// ErrSilentIgnore alerts the runner that the request broke turn order rules and should drop seamlessly.
var ErrSilentIgnore = errors.New("silently ignored by combat rules")

type EventType string

const (
	EventEncounterStarted     EventType = "EncounterStarted"
	EventEncounterEnded       EventType = "EncounterEnded"
	EventActorAdded           EventType = "ActorAdded"
	EventTurnChanged          EventType = "TurnChanged"
	EventHPChanged            EventType = "HPChanged"
	EventDiceRolled           EventType = "DiceRolled"
	EventInitiativeRolled     EventType = "InitiativeRolled"
	EventAttackResolved       EventType = "AttackResolved"
	EventTurnEnded            EventType = "TurnEnded"
	EventHint                 EventType = "Hint"
	EventAskIssued            EventType = "AskIssued"
	EventCheckResolved        EventType = "CheckResolved"
	EventConditionApplied     EventType = "ConditionApplied"
	EventAdjudicationStarted  EventType = "AdjudicationStarted"
	EventAdjudicationResolved EventType = "AdjudicationResolved"
	EventDodgeTaken           EventType = "DodgeTaken"
	EventGrappleTaken         EventType = "GrappleTaken"
	EventActionConsumed       EventType = "ActionConsumed"
	EventHelpTaken            EventType = "HelpTaken"
	EventConditionRemoved     EventType = "ConditionRemoved"
	EventAbilitySpent         EventType = "AbilitySpent"
	EventAbilityRecharged     EventType = "AbilityRecharged"
	EventRechargeRolled       EventType = "RechargeRolled"
)

// Event is the building block of the Event Sourced engine.
type Event interface {
	Type() EventType
	Apply(state *GameState) error
	Message() string
}

// EncounterStartedEvent clears the encounter table.
type EncounterStartedEvent struct{}

func (e *EncounterStartedEvent) Type() EventType { return EventEncounterStarted }
func (e *EncounterStartedEvent) Apply(state *GameState) error {
	state.IsEncounterActive = true
	state.Entities = make(map[string]*Entity)
	state.TurnOrder = make([]string, 0)
	state.Initiatives = make(map[string]int)
	state.CurrentTurn = -1
	return nil
}
func (e *EncounterStartedEvent) Message() string { return "Encounter Started." }

// EncounterEndedEvent drops the active encounter flag
type EncounterEndedEvent struct{}

func (e *EncounterEndedEvent) Type() EventType { return EventEncounterEnded }
func (e *EncounterEndedEvent) Apply(state *GameState) error {
	state.IsEncounterActive = false
	return nil
}
func (e *EncounterEndedEvent) Message() string { return "Encounter Ended." }

// AttackResolvedEvent logs successful and failed strikes across multiple targets
type AttackResolvedEvent struct {
	Attacker      string
	Weapon        string
	Targets       []string
	HitStatus     map[string]bool
	IsOffHand     bool
	IsOpportunity bool
}

func (e *AttackResolvedEvent) Type() EventType { return EventAttackResolved }
func (e *AttackResolvedEvent) Apply(state *GameState) error {
	state.PendingDamage = &PendingDamageState{
		Attacker:  e.Attacker,
		Targets:   e.Targets,
		Weapon:    e.Weapon,
		HitStatus: e.HitStatus,
		IsOffHand: e.IsOffHand,
	}

	if ent, ok := state.Entities[e.Attacker]; ok {
		if e.IsOffHand {
			if ent.BonusActionsRemaining > 0 {
				ent.BonusActionsRemaining--
			}
		} else if e.IsOpportunity {
			if ent.ReactionsRemaining > 0 {
				ent.ReactionsRemaining--
			}
		} else {
			// standard action attack
			ent.HasAttackedThisTurn = true
			ent.LastAttackedWithWeapon = e.Weapon

			if ent.ActionsRemaining > 0 && ent.AttacksRemaining <= 0 {
				ent.ActionsRemaining--
				ent.AttacksRemaining = 1
			}
			ent.AttacksRemaining -= len(e.Targets)
			if ent.AttacksRemaining < 0 {
				ent.AttacksRemaining = 0
			}
		}
	}
	return nil
}

func (e *AttackResolvedEvent) Message() string {
	var sb strings.Builder
	prefix := ""
	if e.IsOffHand {
		prefix = "[OFF-HAND] "
	} else if e.IsOpportunity {
		prefix = "[OPPORTUNITY] "
	}
	sb.WriteString(fmt.Sprintf("%s%s attacks with %s:\n", prefix, e.Attacker, e.Weapon))
	for _, t := range e.Targets {
		status := "Miss!"
		if e.HitStatus[t] {
			status = "Hit!"
		}
		sb.WriteString(fmt.Sprintf("├─ %s: %s\n", t, status))
	}
	return strings.TrimSpace(sb.String())
}

// TurnEndedEvent advances the current turn sequence
type TurnEndedEvent struct {
	ActorID string
}

func (e *TurnEndedEvent) Type() EventType { return EventTurnEnded }
func (e *TurnEndedEvent) Apply(state *GameState) error {
	if len(state.TurnOrder) == 0 {
		return nil
	}

	// Ensure we move exactly from the ending actor's perspective, or just advance 1
	state.CurrentTurn = (state.CurrentTurn + 1) % len(state.TurnOrder)
	return nil
}
func (e *TurnEndedEvent) Message() string {
	return fmt.Sprintf("%s ended its turn.", e.ActorID)
}

// HintEvent is purely for querying the current state, and typically won't be saved to the store
type HintEvent struct {
	MessageStr string
}

func (e *HintEvent) Type() EventType              { return EventHint }
func (e *HintEvent) Apply(state *GameState) error { return nil }
func (e *HintEvent) Message() string              { return e.MessageStr }

// ActorAddedEvent brings a new entity into the encounter tracker.
type ActorAddedEvent struct {
	ID         string
	Category   string // "Character" or "Monster"
	EntityType string // Genre-specific: "undead", etc.
	Name       string
	MaxHP      int
	Stats      map[string]int
	Resources  map[string]int
	Abilities  []data.Ability
}

func (e *ActorAddedEvent) Type() EventType { return EventActorAdded }
func (e *ActorAddedEvent) Apply(state *GameState) error {
	if _, ok := state.Entities[e.ID]; ok {
		return fmt.Errorf("actor with ID %s already tracking in encounter", e.ID)
	}

	state.Entities[e.ID] = &Entity{
		ID:                    e.ID,
		Category:              e.Category,
		EntityType:            e.EntityType,
		Name:                  e.Name,
		HP:                    e.MaxHP,
		MaxHP:                 e.MaxHP,
		Stats:                 e.Stats,
		Resources:             e.Resources,
		Abilities:             e.Abilities,
		ActionsRemaining:      1,
		BonusActionsRemaining: 1,
		ReactionsRemaining:    1,
		AttacksRemaining:      1,
	}
	state.TurnOrder = append(state.TurnOrder, e.ID)
	return nil
}
func (e *ActorAddedEvent) Message() string {
	return fmt.Sprintf("Added actor %s (%d/%d HP)", e.Name, e.MaxHP, e.MaxHP)
}

// TurnChangedEvent advances the current turn marker.
type TurnChangedEvent struct {
	ActorID string
}

func (e *TurnChangedEvent) Type() EventType { return EventTurnChanged }
func (e *TurnChangedEvent) Apply(state *GameState) error {
	// Simple lookup for turn index
	for i, id := range state.TurnOrder {
		if id == e.ActorID {
			state.CurrentTurn = i

			// Reset Action Economy for the actor starting their turn
			if ent, ok := state.Entities[e.ActorID]; ok {
				ent.ActionsRemaining = 1
				ent.BonusActionsRemaining = 1
				ent.ReactionsRemaining = 1
				ent.AttacksRemaining = 1 // Basic assumption, will be overridden by stat load later if needed

				ent.HasAttackedThisTurn = false
				ent.LastAttackedWithWeapon = ""

				// Remove "Dodging" condition
				newConds := []string{}
				for _, c := range ent.Conditions {
					if c != "Dodging" {
						newConds = append(newConds, c)
					}
				}
				ent.Conditions = newConds

				// Expire any Help benefits provided by this actor
				for _, entity := range state.Entities {
					activeConds := []string{}
					for _, c := range entity.Conditions {
						isHelp := strings.HasPrefix(c, "HelpedCheck:") || strings.HasPrefix(c, "HelpedAttack:")
						if isHelp {
							parts := strings.Split(c, ":")
							if len(parts) == 2 && parts[1] == e.ActorID {
								continue // Expired
							}
						}
						activeConds = append(activeConds, c)
					}
					entity.Conditions = activeConds
				}
			}
			return nil
		}
	}
	return fmt.Errorf("actor %s not found in turn order", e.ActorID)
}
func (e *TurnChangedEvent) Message() string { return fmt.Sprintf("Turn changed to %s", e.ActorID) }

// HPChangedEvent modifies an actor's current HP (positive heals, negative damages).
type HPChangedEvent struct {
	ActorID string
	Amount  int
}

func (e *HPChangedEvent) Type() EventType { return EventHPChanged }
func (e *HPChangedEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("actor %s not found", e.ActorID)
	}

	ent.HP += e.Amount
	if ent.HP < 0 {
		ent.HP = 0
	}
	if ent.HP > ent.MaxHP {
		ent.HP = ent.MaxHP
	}
	state.PendingDamage = nil // clear pending damage after resolution
	return nil
}
func (e *HPChangedEvent) Message() string {
	if e.Amount > 0 {
		return fmt.Sprintf("%s healed for %d HP", e.ActorID, e.Amount)
	} else if e.Amount < 0 {
		return fmt.Sprintf("%s took %d damage", e.ActorID, -e.Amount)
	}
	return fmt.Sprintf("%s HP was unchanged", e.ActorID)
}

// DiceRolledEvent tracks a dice roll macro result.
type DiceRolledEvent struct {
	ActorName string
	Total     int
	RawRolls  []int
	Kept      []int
	Dropped   []int
	Modifier  int
}

func (e *DiceRolledEvent) Type() EventType { return EventDiceRolled }
func (e *DiceRolledEvent) Apply(state *GameState) error {
	return nil // Dice rolls do not inherently modify state
}
func (e *DiceRolledEvent) Message() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s rolled: %d\n", e.ActorName, e.Total))
	sb.WriteString(fmt.Sprintf("├─ Raw: %v\n", e.RawRolls))

	if len(e.Dropped) > 0 {
		sb.WriteString(fmt.Sprintf("├─ Kept: %v\n", e.Kept))
		sb.WriteString(fmt.Sprintf("├─ Dropped: %v\n", e.Dropped))
	}
	if e.Modifier != 0 {
		modPrefix := "+"
		if e.Modifier > 0 {
			modPrefix = ""
		}
		sb.WriteString(fmt.Sprintf("├─ Modifier: %s%d\n", modPrefix, e.Modifier))
	}
	return strings.TrimSpace(sb.String())
}

// InitiativeRolledEvent stores a participant's initiative roll and re-sorts turn order
type InitiativeRolledEvent struct {
	ActorName string
	Score     int
	IsManual  bool
	RawRolls  []int
	Kept      []int
	Dropped   []int
	Modifier  int
}

func (e *InitiativeRolledEvent) Type() EventType { return EventInitiativeRolled }
func (e *InitiativeRolledEvent) Apply(state *GameState) error {
	state.Initiatives[e.ActorName] = e.Score

	// Create fresh sorted TurnOrder whenever new initiative arrives
	var names []string
	for id := range state.Entities {
		names = append(names, id)
	}
	sort.SliceStable(names, func(i, j int) bool {
		scoreI, okI := state.Initiatives[names[i]]
		scoreJ, okJ := state.Initiatives[names[j]]
		if okI && okJ {
			return scoreI > scoreJ
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return names[i] < names[j] // tie-break by ID for stability
	})

	// Safely preserve CurrentTurn actor across resort if possible
	var currentActor string
	if state.CurrentTurn >= 0 && state.CurrentTurn < len(state.TurnOrder) {
		currentActor = state.TurnOrder[state.CurrentTurn]
	}

	state.TurnOrder = names

	// Realign index
	if currentActor != "" {
		found := false
		for i, name := range state.TurnOrder {
			if name == currentActor {
				state.CurrentTurn = i
				found = true
				break
			}
		}
		if !found {
			// If our actor disappeared, fallback to top
			state.CurrentTurn = 0
		}
	} else {
		// If we didn't have a turn yet, check if we've fulfilled initiative requirements
		isNowFrozen := false
		for id := range state.Entities {
			if _, ok := state.Initiatives[id]; !ok {
				isNowFrozen = true
				break
			}
		}
		if !isNowFrozen {
			state.CurrentTurn = 0
		}
	}
	return nil
}
func (e *InitiativeRolledEvent) Message() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s rolled initiative: %d", e.ActorName, e.Score))

	if e.IsManual {
		sb.WriteString("\n├─ (Manual Override)")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("\n├─ Raw: %v", e.RawRolls))

	if len(e.Dropped) > 0 {
		sb.WriteString(fmt.Sprintf("\n├─ Kept: %v", e.Kept))
		sb.WriteString(fmt.Sprintf("\n├─ Dropped: %v", e.Dropped))
	}
	if e.Modifier != 0 {
		modPrefix := "+"
		if e.Modifier > 0 {
			modPrefix = ""
		}
		sb.WriteString(fmt.Sprintf("\n├─ Modifier: %s%d", modPrefix, e.Modifier))
	}
	return sb.String()
}

// AskIssuedEvent freezes target actors to roll a specific check
type AskIssuedEvent struct {
	Targets  []string         `json:"targets"`
	Check    []string         `json:"check"`
	DC       int              `json:"dc"`
	Fails    *RollConsequence `json:"fails"`
	Succeeds *RollConsequence `json:"succeeds"`
}

func (e *AskIssuedEvent) Type() EventType { return EventAskIssued }
func (e *AskIssuedEvent) Apply(state *GameState) error {
	for _, t := range e.Targets {
		state.PendingChecks[t] = &PendingCheckState{
			Check:    e.Check,
			DC:       e.DC,
			Fails:    e.Fails,
			Succeeds: e.Succeeds,
		}
	}
	return nil
}
func (e *AskIssuedEvent) Message() string {
	return fmt.Sprintf("GM asked for %v check (DC %d).", e.Check, e.DC)
}

// CheckResolvedEvent marks the fulfillment of a required check
type CheckResolvedEvent struct {
	ActorID string `json:"actor_id"`
	Result  int    `json:"result"`
	Success bool   `json:"success"`
}

func (e *CheckResolvedEvent) Type() EventType { return EventCheckResolved }
func (e *CheckResolvedEvent) Apply(state *GameState) error {
	delete(state.PendingChecks, e.ActorID)
	return nil
}
func (e *CheckResolvedEvent) Message() string {
	resolution := "failed"
	if e.Success {
		resolution = "succeeded"
	}
	return fmt.Sprintf("%s rolled %d and %s the check.", e.ActorID, e.Result, resolution)
}

// ConditionAppliedEvent adds a condition to an actor
type ConditionAppliedEvent struct {
	ActorID   string `json:"actor_id"`
	Condition string `json:"condition"`
}

func (e *ConditionAppliedEvent) Type() EventType { return EventConditionApplied }
func (e *ConditionAppliedEvent) Apply(state *GameState) error {
	if ent, ok := state.Entities[e.ActorID]; ok {
		hasIt := false
		for _, c := range ent.Conditions {
			if c == e.Condition {
				hasIt = true
				break
			}
		}
		if !hasIt {
			ent.Conditions = append(ent.Conditions, e.Condition)
		}
	}
	return nil
}
func (e *ConditionAppliedEvent) Message() string {
	return fmt.Sprintf("%s is now %s.", e.ActorID, e.Condition)
}

// AdjudicationStartedEvent freezes the system for GM authorization
type AdjudicationStartedEvent struct {
	OriginalCommand string
}

func (e *AdjudicationStartedEvent) Type() EventType { return EventAdjudicationStarted }
func (e *AdjudicationStartedEvent) Apply(state *GameState) error {
	state.PendingAdjudication = &PendingAdjudicationState{
		OriginalCommand: e.OriginalCommand,
	}
	return nil
}
func (e *AdjudicationStartedEvent) Message() string {
	return fmt.Sprintf("Adjudicate \"%s\"", e.OriginalCommand)
}

// AdjudicationResolvedEvent records the GM decision
type AdjudicationResolvedEvent struct {
	Allowed bool
}

func (e *AdjudicationResolvedEvent) Type() EventType { return EventAdjudicationResolved }
func (e *AdjudicationResolvedEvent) Apply(state *GameState) error {
	if e.Allowed {
		if state.PendingAdjudication != nil {
			state.PendingAdjudication.Approved = true
		}
	} else {
		state.PendingAdjudication = nil
	}
	return nil
}
func (e *AdjudicationResolvedEvent) Message() string {
	if e.Allowed {
		return "GM allowed the action."
	}
	return "GM denied the action."
}

// DodgeTakenEvent records that an actor is dodging
type DodgeTakenEvent struct {
	ActorID string
}

func (e *DodgeTakenEvent) Type() EventType { return EventDodgeTaken }
func (e *DodgeTakenEvent) Apply(state *GameState) error {
	if ent, ok := state.Entities[e.ActorID]; ok {
		if ent.ActionsRemaining > 0 {
			ent.ActionsRemaining--
		}

		hasIt := false
		for _, c := range ent.Conditions {
			if c == "Dodging" {
				hasIt = true
				break
			}
		}
		if !hasIt {
			ent.Conditions = append(ent.Conditions, "Dodging")
		}
	}
	return nil
}
func (e *DodgeTakenEvent) Message() string {
	return fmt.Sprintf("%s is now Dodging.", e.ActorID)
}

// GrappleTakenEvent records a grapple attempt
type GrappleTakenEvent struct {
	Attacker string
	Target   string
}

func (e *GrappleTakenEvent) Type() EventType { return EventGrappleTaken }
func (e *GrappleTakenEvent) Apply(state *GameState) error {
	// Clears adjudication as the action has now resolved into its consequences (Ask)
	state.PendingAdjudication = nil
	return nil
}
func (e *GrappleTakenEvent) Message() string {
	return fmt.Sprintf("%s attempts to grapple %s.", e.Attacker, e.Target)
}

// ActionConsumedEvent record usage of a standard action
type ActionConsumedEvent struct {
	ActorID string
}

func (e *ActionConsumedEvent) Type() EventType { return EventActionConsumed }
func (e *ActionConsumedEvent) Apply(state *GameState) error {
	if ent, ok := state.Entities[e.ActorID]; ok {
		if ent.ActionsRemaining > 0 {
			ent.ActionsRemaining--
		}
	}
	// Once an action is consumed, any pending adjudication for it is also resolved/finished
	state.PendingAdjudication = nil
	return nil
}
func (e *ActionConsumedEvent) Message() string {
	return fmt.Sprintf("%s used an action.", e.ActorID)
}

// HelpTakenEvent records a help action was performed
type HelpTakenEvent struct {
	HelperID string
	TargetID string
	HelpType string // "check" or "attack"
}

func (e *HelpTakenEvent) Type() EventType { return EventHelpTaken }
func (e *HelpTakenEvent) Apply(state *GameState) error {
	if ent, ok := state.Entities[e.TargetID]; ok {
		// Capitalize first letter
		uType := strings.ToUpper(e.HelpType[0:1]) + e.HelpType[1:]
		condition := fmt.Sprintf("Helped%s:%s", uType, e.HelperID)
		ent.Conditions = append(ent.Conditions, condition)
	}
	if ent, ok := state.Entities[e.HelperID]; ok {
		if ent.ActionsRemaining > 0 {
			ent.ActionsRemaining--
		}
	}
	state.PendingAdjudication = nil
	return nil
}
func (e *HelpTakenEvent) Message() string {
	return fmt.Sprintf("%s helps %s with an %s.", e.HelperID, e.TargetID, e.HelpType)
}

// ConditionRemovedEvent forcibly removes a condition
type ConditionRemovedEvent struct {
	ActorID   string
	Condition string
}

func (e *ConditionRemovedEvent) Type() EventType { return EventConditionRemoved }
func (e *ConditionRemovedEvent) Apply(state *GameState) error {
	if ent, ok := state.Entities[e.ActorID]; ok {
		newConds := []string{}
		for _, c := range ent.Conditions {
			if c != e.Condition {
				newConds = append(newConds, c)
			}
		}
		ent.Conditions = newConds
	}
	return nil
}
func (e *ConditionRemovedEvent) Message() string {
	return fmt.Sprintf("%s is no longer %s.", e.ActorID, e.Condition)
}

// AbilitySpentEvent marks a monster ability as cooling down
type AbilitySpentEvent struct {
	ActorID    string
	ActionName string
}

func (e *AbilitySpentEvent) Type() EventType { return EventAbilitySpent }
func (e *AbilitySpentEvent) Apply(state *GameState) error {
	state.SpentRecharges[e.ActorID] = append(state.SpentRecharges[e.ActorID], e.ActionName)
	return nil
}
func (e *AbilitySpentEvent) Message() string {
	return fmt.Sprintf("%s spent %s (cooling down).", e.ActorID, e.ActionName)
}

// AbilityRechargedEvent marks a monster ability as available again
type AbilityRechargedEvent struct {
	ActorID    string
	ActionName string
}

func (e *AbilityRechargedEvent) Type() EventType { return EventAbilityRecharged }
func (e *AbilityRechargedEvent) Apply(state *GameState) error {
	spent := state.SpentRecharges[e.ActorID]
	newSpent := []string{}
	for _, s := range spent {
		if s != e.ActionName {
			newSpent = append(newSpent, s)
		}
	}
	state.SpentRecharges[e.ActorID] = newSpent
	return nil
}
func (e *AbilityRechargedEvent) Message() string {
	return fmt.Sprintf("%s's %s recharged!", e.ActorID, e.ActionName)
}

// RechargeRolledEvent records the attempt to recharge an ability
type RechargeRolledEvent struct {
	ActorID     string
	ActionName  string
	Roll        int
	Requirement string
	Success     bool
}

func (e *RechargeRolledEvent) Type() EventType { return EventRechargeRolled }
func (e *RechargeRolledEvent) Apply(state *GameState) error {
	return nil // Purely informational
}
func (e *RechargeRolledEvent) Message() string {
	resolution := "Failed"
	if e.Success {
		resolution = "Success!"
	}
	return fmt.Sprintf("Recharge %s for %s: Rolled %d (Req: %s) -> %s", e.ActionName, e.ActorID, e.Roll, e.Requirement, resolution)
}
