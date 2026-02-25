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
	EventConditionRemoved     EventType = "ConditionRemoved"
	EventAbilitySpent         EventType = "AbilitySpent"
	EventAbilityRecharged     EventType = "AbilityRecharged"
	EventRechargeRolled       EventType = "RechargeRolled"

	// Truly Generic Events
	EventAttributeChanged   EventType = "AttributeChanged"
	EventConditionToggled   EventType = "ConditionToggled"
	EventMetadataChanged    EventType = "MetadataChanged"
	EventFrozenUntilChanged EventType = "FrozenUntilChanged"
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
	state.Metadata = make(map[string]any)
	state.CurrentTurn = -1
	state.FrozenUntil = ""
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
	state.Metadata["pending_damage"] = map[string]any{
		"attacker":    e.Attacker,
		"targets":     e.Targets,
		"weapon":      e.Weapon,
		"hit_status":  e.HitStatus,
		"is_off_hand": e.IsOffHand,
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
	ID            string
	Category      string // "Character" or "Monster"
	EntityType    string // Genre-specific: "undead", etc.
	Name          string
	Size          string
	MaxHP         int
	Stats         map[string]int
	Resources     map[string]int
	Abilities     []data.Ability
	Proficiencies map[string]int
	Defenses      []data.Defense // TODO: Genericize defenses
}

func (e *ActorAddedEvent) Type() EventType { return EventActorAdded }
func (e *ActorAddedEvent) Apply(state *GameState) error {
	if _, ok := state.Entities[e.ID]; ok {
		return fmt.Errorf("actor with ID %s already tracking in encounter", e.ID)
	}

	state.Entities[e.ID] = &Entity{
		ID:            e.ID,
		Name:          e.Name,
		Types:         []string{e.EntityType},
		Classes:       map[string]string{"size": e.Size, "category": e.Category},
		Stats:         e.Stats,
		Resources:     e.Resources,
		Spent:         make(map[string]int),
		Conditions:    []string{},
		Proficiencies: e.Proficiencies,
		Statuses:      make(map[string]string),
		Inventory:     make(map[string]int),
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
	for i, id := range state.TurnOrder {
		if id == e.ActorID {
			state.CurrentTurn = i
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

	// In the generic model, "hp" is just a resource.
	// We decrement Spent[hp] for healing, and increment it for damage.
	// But wait, the math in HPChangedEvent used positive for heals and negative for damages.
	// So: newSpent = oldSpent - healAmount, or newSpent = oldSpent + damageAmount.
	// e.Amount is positive for heals, negative for damages.
	// So: newSpent = oldSpent - e.Amount.
	if ent.Spent == nil {
		ent.Spent = make(map[string]int)
	}
	ent.Spent["hp"] -= e.Amount
	if ent.Spent["hp"] < 0 {
		ent.Spent["hp"] = 0
	}
	maxHP := ent.Resources["hp"]
	if ent.Spent["hp"] > maxHP {
		ent.Spent["hp"] = maxHP
	}

	delete(state.Metadata, "pending_damage")
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
	initiatives, ok := state.Metadata["initiatives"].(map[string]int)
	if !ok {
		// Try to recover if it was saved as map[string]any by JSON
		if m, ok := state.Metadata["initiatives"].(map[string]any); ok {
			initiatives = make(map[string]int)
			for k, v := range m {
				if vi, ok := v.(int); ok {
					initiatives[k] = vi
				} else if vf, ok := v.(float64); ok {
					initiatives[k] = int(vf)
				}
			}
		} else {
			initiatives = make(map[string]int)
		}
		state.Metadata["initiatives"] = initiatives
	}
	initiatives[e.ActorName] = e.Score

	// Create fresh sorted TurnOrder whenever new initiative arrives
	var names []string
	for id := range state.Entities {
		names = append(names, id)
	}
	sort.SliceStable(names, func(i, j int) bool {
		scoreI, okI := initiatives[names[i]]
		scoreJ, okJ := initiatives[names[j]]
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
			state.CurrentTurn = 0
		}
	} else {
		// If we didn't have a turn yet, check if we've fulfilled initiative requirements
		isNowFrozen := false
		for id := range state.Entities {
			if _, ok := initiatives[id]; !ok {
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
	Targets  []string       `json:"targets"`
	Check    []string       `json:"check"`
	DC       int            `json:"dc"`
	Fails    map[string]any `json:"fails"`
	Succeeds map[string]any `json:"succeeds"`
}

func (e *AskIssuedEvent) Type() EventType { return EventAskIssued }
func (e *AskIssuedEvent) Apply(state *GameState) error {
	pendingChecks, ok := state.Metadata["pending_checks"].(map[string]any)
	if !ok {
		pendingChecks = make(map[string]any)
		state.Metadata["pending_checks"] = pendingChecks
	}

	for _, t := range e.Targets {
		pendingChecks[t] = map[string]any{
			"check":    e.Check,
			"dc":       e.DC,
			"fails":    e.Fails,
			"succeeds": e.Succeeds,
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
	if pendingChecks, ok := state.Metadata["pending_checks"].(map[string]any); ok {
		delete(pendingChecks, e.ActorID)
	}
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
	ActorID        string `json:"actor_id"`
	Condition      string `json:"condition"`
	ExpiresOn      string `json:"expires_on"`
	ReferenceActor string `json:"reference_actor"`
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

		if e.ExpiresOn != "" && e.ReferenceActor != "" {
			expMap, ok := state.Metadata["conditions_expiry"].(map[string]any)
			if !ok {
				expMap = make(map[string]any)
				state.Metadata["conditions_expiry"] = expMap
			}
			expMap[fmt.Sprintf("%s:%s", e.ActorID, e.Condition)] = map[string]string{
				"expires_on":      e.ExpiresOn,
				"reference_actor": e.ReferenceActor,
			}
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
	state.Metadata["pending_adjudication"] = map[string]any{
		"original_command": e.OriginalCommand,
		"approved":         false,
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
		if adj, ok := state.Metadata["pending_adjudication"].(map[string]any); ok {
			adj["approved"] = true
		}
	} else {
		delete(state.Metadata, "pending_adjudication")
	}
	return nil
}
func (e *AdjudicationResolvedEvent) Message() string {
	if e.Allowed {
		return "GM allowed the action."
	}
	return "GM denied the action."
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
	spentRecharges, ok := state.Metadata["spent_recharges"].(map[string][]string)
	if !ok {
		spentRecharges = make(map[string][]string)
		state.Metadata["spent_recharges"] = spentRecharges
	}
	spentRecharges[e.ActorID] = append(spentRecharges[e.ActorID], e.ActionName)
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
	spentRecharges, ok := state.Metadata["spent_recharges"].(map[string][]string)
	if !ok {
		return nil
	}
	spent := spentRecharges[e.ActorID]
	newSpent := []string{}
	for _, s := range spent {
		if s != e.ActionName {
			newSpent = append(newSpent, s)
		}
	}
	spentRecharges[e.ActorID] = newSpent
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

// AttributeType classifies what kind of data is being changed
type AttributeType string

const (
	AttrStat         AttributeType = "stat"
	AttrResource     AttributeType = "resource"
	AttrSpent        AttributeType = "spent"
	AttrStatus       AttributeType = "status"
	AttrClass        AttributeType = "class"
	AttrType         AttributeType = "type"
	AttrProficiency  AttributeType = "proficiency"
	AttrInventory    AttributeType = "inventory"
	AttrActionPoints AttributeType = "action_points" // e.g. "actions", "bonus_actions"
)

// AttributeChangedEvent updates a specific key in an entity's generic data maps
type AttributeChangedEvent struct {
	ActorID  string        `json:"actor_id"`
	AttrType AttributeType `json:"attr_type"`
	Key      string        `json:"key"`
	Value    any           `json:"value"`
	OldValue any           `json:"old_value"`
}

func (e *AttributeChangedEvent) Type() EventType { return EventAttributeChanged }
func (e *AttributeChangedEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("actor %s not found", e.ActorID)
	}

	switch e.AttrType {
	case AttrStat:
		if v, ok := e.Value.(int); ok {
			ent.Stats[e.Key] = v
		}
	case AttrResource:
		if v, ok := e.Value.(int); ok {
			ent.Resources[e.Key] = v
		}
	case AttrSpent:
		if v, ok := e.Value.(int); ok {
			ent.Spent[e.Key] = v
		}
	case AttrStatus:
		if v, ok := e.Value.(string); ok {
			ent.Statuses[e.Key] = v
		}
	case AttrClass:
		if v, ok := e.Value.(string); ok {
			ent.Classes[e.Key] = v
		}
	case AttrType:
		if v, ok := e.Value.(string); ok {
			// Find and replace or just append? Proposed was []string.
			// For simplicity in this event, we'll treat it as a replace or append logic.
			// Actually, let's keep it simple: Types is a slice.
			found := false
			for _, t := range ent.Types {
				if t == v {
					found = true
					break
				}
			}
			if !found {
				ent.Types = append(ent.Types, v)
			}
		}
	case AttrProficiency:
		if v, ok := e.Value.(int); ok {
			ent.Proficiencies[e.Key] = v
		}
	case AttrInventory:
		if v, ok := e.Value.(int); ok {
			ent.Inventory[e.Key] = v
		}
	}
	return nil
}
func (e *AttributeChangedEvent) Message() string {
	return fmt.Sprintf("%s's %s %s changed to %v.", e.ActorID, e.AttrType, e.Key, e.Value)
}

// ConditionToggledEvent adds or removes a condition from an actor
type ConditionToggledEvent struct {
	ActorID   string `json:"actor_id"`
	Condition string `json:"condition"`
	Active    bool   `json:"active"`
}

func (e *ConditionToggledEvent) Type() EventType { return EventConditionToggled }
func (e *ConditionToggledEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("actor %s not found", e.ActorID)
	}

	if e.Active {
		found := false
		for _, c := range ent.Conditions {
			if c == e.Condition {
				found = true
				break
			}
		}
		if !found {
			ent.Conditions = append(ent.Conditions, e.Condition)
		}
	} else {
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
func (e *ConditionToggledEvent) Message() string {
	if e.Active {
		return fmt.Sprintf("%s is now %s.", e.ActorID, e.Condition)
	}
	return fmt.Sprintf("%s is no longer %s.", e.ActorID, e.Condition)
}

// MetadataChangedEvent updates a key in the global state metadata map
type MetadataChangedEvent struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func (e *MetadataChangedEvent) Type() EventType { return EventMetadataChanged }
func (e *MetadataChangedEvent) Apply(state *GameState) error {
	if e.Value == nil {
		delete(state.Metadata, e.Key)
	} else {
		state.Metadata[e.Key] = e.Value
	}
	return nil
}
func (e *MetadataChangedEvent) Message() string {
	return fmt.Sprintf("System metadata %s updated.", e.Key)
}

// FrozenUntilChangedEvent updates the freeze condition of the engine
type FrozenUntilChangedEvent struct {
	FrozenUntil string `json:"frozen_until"`
}

func (e *FrozenUntilChangedEvent) Type() EventType { return EventFrozenUntilChanged }
func (e *FrozenUntilChangedEvent) Apply(state *GameState) error {
	state.FrozenUntil = e.FrozenUntil
	return nil
}
func (e *FrozenUntilChangedEvent) Message() string {
	if e.FrozenUntil == "" {
		return "Engine unfrozen."
	}
	return fmt.Sprintf("Engine frozen until: %s", e.FrozenUntil)
}
