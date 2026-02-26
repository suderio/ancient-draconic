// Package manifest implements the manifest-driven command engine.
// It provides a clean-slate, system-agnostic execution pipeline
// where game rules are defined entirely in YAML manifests.
package engine

import "fmt"

// --- Manifest model ---

// ParamDef declares a named parameter for a command with its type and optionality.
// Supported types: "string", "int", "target", "list<target>".
type ParamDef struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
}

// PrereqStep defines a prerequisite check that must pass before a command executes.
// If the Formula evaluates to false, the Error message is returned to the caller.
type PrereqStep struct {
	Name    string `yaml:"name"`
	Formula string `yaml:"formula"`
	Error   string `yaml:"error"`
}

// GameStep defines a single evaluation step in a command's execution.
// The Formula is a CEL expression evaluated against the current context.
// The Event field names the engine event to emit with the formula's result.
type GameStep struct {
	Name    string `yaml:"name"`
	Formula string `yaml:"formula"`
	Event   string `yaml:"event"`
	Loop    string `yaml:"loop"` // Optional: target loop name for loop events (defaults to cmdName)
}

// CommandDef is the structured definition of a manifest-driven command.
// It separates concerns into distinct phases: parameter validation, prerequisites,
// game logic, per-target logic, and actor-affecting logic.
type CommandDef struct {
	Name    string       `yaml:"name"`
	Params  []ParamDef   `yaml:"params"`
	Prereq  []PrereqStep `yaml:"prereq"`
	Hint    string       `yaml:"hint"`
	Help    string       `yaml:"help"`
	Error   string       `yaml:"error"` // Usage string shown on invalid input
	Game    []GameStep   `yaml:"game"`
	Targets []GameStep   `yaml:"targets"`
	Actor   []GameStep   `yaml:"actor"`
}

// Restrictions defines cross-cutting rules that apply to multiple commands.
type Restrictions struct {
	Adjudication struct {
		Commands []string `yaml:"commands"`
	} `yaml:"adjudication"`
	GMCommands []string `yaml:"gm_commands"`
}

// Manifest is the top-level structure of a campaign manifest YAML file.
// It contains all command definitions and cross-cutting restrictions.
type Manifest struct {
	Restrictions Restrictions          `yaml:"restrictions"`
	Commands     map[string]CommandDef `yaml:"commands"`
}

// --- Entity model ---

// Entity represents an actor or target participating in the game.
// This is the canonical model for characters, monsters, and any other
// tracked game entity. YAML character/monster files deserialize into this.
type Entity struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name" yaml:"name"`
	Types         []string          `json:"types" yaml:"types"`                 // e.g., "monster", "undead"
	Classes       map[string]string `json:"classes" yaml:"classes"`             // e.g., "size": "medium"
	Stats         map[string]int    `json:"stats" yaml:"stats"`                 // e.g., "str": 16
	Resources     map[string]int    `json:"resources" yaml:"resources"`         // max values (e.g., "hp": 20)
	Spent         map[string]int    `json:"spent" yaml:"spent"`                 // current usage (e.g., "hp": 5)
	Conditions    []string          `json:"conditions" yaml:"conditions"`       // e.g., "poisoned"
	Proficiencies map[string]int    `json:"proficiencies" yaml:"proficiencies"` // e.g., "athletics": 2
	Statuses      map[string]string `json:"statuses" yaml:"statuses"`           // e.g., "concentrating": "true"
	Inventory     map[string]int    `json:"inventory" yaml:"inventory"`         // items and counts
}

// NewEntity creates an Entity with all maps initialized to avoid nil-map panics.
func NewEntity(id, name string) *Entity {
	return &Entity{
		ID:            id,
		Name:          name,
		Types:         make([]string, 0),
		Classes:       make(map[string]string),
		Stats:         make(map[string]int),
		Resources:     make(map[string]int),
		Spent:         make(map[string]int),
		Conditions:    make([]string, 0),
		Proficiencies: make(map[string]int),
		Statuses:      make(map[string]string),
		Inventory:     make(map[string]int),
	}
}

// --- Game state ---

// Loop represents an ordered sequence of actors taking turns (e.g., combat encounter).
// Actors are sorted by their Order value; Ascending controls sort direction.
type Loop struct {
	Active    bool           `json:"active"`
	Actors    []string       `json:"actors"`
	Order     map[string]int `json:"order"` // actor ID → sort key
	Ascending bool           `json:"ascending"`
	Current   int            `json:"current"` // index into sorted actor list
}

// GameState is the full projection of game state, built from applied events.
type GameState struct {
	Entities map[string]*Entity `json:"entities"`
	Loops    map[string]*Loop   `json:"loops"`
	Metadata map[string]any     `json:"metadata"`

	// LastCommand tracks the name of the last successfully executed command,
	// used by the "hint" hardcoded command.
	LastCommand string `json:"last_command"`
}

// NewGameState creates a clean, empty game state with all maps initialized.
func NewGameState() *GameState {
	return &GameState{
		Entities: make(map[string]*Entity),
		Loops:    make(map[string]*Loop),
		Metadata: make(map[string]any),
	}
}

// IsLoopActive returns whether a named loop is currently active.
func (s *GameState) IsLoopActive(name string) bool {
	if l, ok := s.Loops[name]; ok {
		return l.Active
	}
	return false
}

// --- Events ---

// Event is the building block of the event-sourced engine.
// Every state change is represented as an Event that can be applied to GameState.
type Event interface {
	Type() string
	Apply(state *GameState) error
	Message() string
}

// LoopEvent starts or stops a named loop (e.g., "encounter").
// When Active is true, the loop is created/activated.
// When Active is false, it is deactivated and its actors are cleared.
type LoopEvent struct {
	LoopName string `json:"loop_name"`
	Active   bool   `json:"active"`
}

func (e *LoopEvent) Type() string { return "LoopEvent" }
func (e *LoopEvent) Apply(state *GameState) error {
	if e.Active {
		state.Loops[e.LoopName] = &Loop{
			Active:  true,
			Actors:  make([]string, 0),
			Order:   make(map[string]int),
			Current: 0,
		}
	} else {
		if l, ok := state.Loops[e.LoopName]; ok {
			l.Active = false
		}
	}
	return nil
}
func (e *LoopEvent) Message() string {
	if e.Active {
		return e.LoopName + " started"
	}
	return e.LoopName + " ended"
}

// LoopOrderAscendingEvent sets whether a loop sorts actors in ascending order.
type LoopOrderAscendingEvent struct {
	LoopName  string `json:"loop_name"`
	Ascending bool   `json:"ascending"`
}

func (e *LoopOrderAscendingEvent) Type() string { return "LoopOrderAscendingEvent" }
func (e *LoopOrderAscendingEvent) Apply(state *GameState) error {
	if l, ok := state.Loops[e.LoopName]; ok {
		l.Ascending = e.Ascending
	}
	return nil
}
func (e *LoopOrderAscendingEvent) Message() string { return "" }

// LoopOrderEvent sets an actor's sort key within a loop (e.g., initiative score).
type LoopOrderEvent struct {
	LoopName string `json:"loop_name"`
	ActorID  string `json:"actor_id"`
	Value    int    `json:"value"`
}

func (e *LoopOrderEvent) Type() string { return "LoopOrderEvent" }
func (e *LoopOrderEvent) Apply(state *GameState) error {
	if l, ok := state.Loops[e.LoopName]; ok {
		l.Order[e.ActorID] = e.Value
	}
	return nil
}
func (e *LoopOrderEvent) Message() string {
	return e.ActorID + " order set"
}

// ActorAddedEvent adds an actor to a named loop.
type ActorAddedEvent struct {
	LoopName string `json:"loop_name"`
	ActorID  string `json:"actor_id"`
}

func (e *ActorAddedEvent) Type() string { return "ActorAddedEvent" }
func (e *ActorAddedEvent) Apply(state *GameState) error {
	if l, ok := state.Loops[e.LoopName]; ok {
		// Avoid duplicates
		for _, a := range l.Actors {
			if a == e.ActorID {
				return nil
			}
		}
		l.Actors = append(l.Actors, e.ActorID)
	}
	return nil
}
func (e *ActorAddedEvent) Message() string {
	return e.ActorID + " added to " + e.LoopName
}

// AttributeChangedEvent modifies a specific field in an entity's data maps.
// Section determines which map to update: "stats", "resources", "spent",
// "statuses", "classes", "inventory".
type AttributeChangedEvent struct {
	ActorID string `json:"actor_id"`
	Section string `json:"section"` // "stats", "resources", "spent", "statuses", "classes", "inventory"
	Key     string `json:"key"`
	Value   any    `json:"value"`
}

func (e *AttributeChangedEvent) Type() string { return "AttributeChangedEvent" }
func (e *AttributeChangedEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("entity %s not found", e.ActorID)
	}
	switch e.Section {
	case "stats":
		if v, ok := toInt(e.Value); ok {
			ent.Stats[e.Key] = v
		}
	case "resources":
		if v, ok := toInt(e.Value); ok {
			ent.Resources[e.Key] = v
		}
	case "spent":
		if v, ok := toInt(e.Value); ok {
			ent.Spent[e.Key] = v
		}
	case "statuses":
		if v, ok := e.Value.(string); ok {
			ent.Statuses[e.Key] = v
		}
	case "classes":
		if v, ok := e.Value.(string); ok {
			ent.Classes[e.Key] = v
		}
	case "inventory":
		if v, ok := toInt(e.Value); ok {
			ent.Inventory[e.Key] = v
		}
	}
	return nil
}
func (e *AttributeChangedEvent) Message() string {
	return fmt.Sprintf("%s.%s.%s changed", e.ActorID, e.Section, e.Key)
}

// AddSpentEvent is a shorthand that increments entity.Spent[Key] by 1.
// This is the common case for consuming actions, reactions, etc.
type AddSpentEvent struct {
	ActorID string `json:"actor_id"`
	Key     string `json:"key"`
}

func (e *AddSpentEvent) Type() string { return "AddSpentEvent" }
func (e *AddSpentEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("entity %s not found", e.ActorID)
	}
	ent.Spent[e.Key]++
	return nil
}
func (e *AddSpentEvent) Message() string {
	return fmt.Sprintf("%s spent %s", e.ActorID, e.Key)
}

// ConditionEvent adds or removes a condition from an entity.
type ConditionEvent struct {
	ActorID   string `json:"actor_id"`
	Condition string `json:"condition"`
	Add       bool   `json:"add"`
}

func (e *ConditionEvent) Type() string { return "ConditionEvent" }
func (e *ConditionEvent) Apply(state *GameState) error {
	ent, ok := state.Entities[e.ActorID]
	if !ok {
		return fmt.Errorf("entity %s not found", e.ActorID)
	}
	if e.Add {
		// Avoid duplicates
		for _, c := range ent.Conditions {
			if c == e.Condition {
				return nil
			}
		}
		ent.Conditions = append(ent.Conditions, e.Condition)
	} else {
		for i, c := range ent.Conditions {
			if c == e.Condition {
				ent.Conditions = append(ent.Conditions[:i], ent.Conditions[i+1:]...)
				break
			}
		}
	}
	return nil
}
func (e *ConditionEvent) Message() string {
	if e.Add {
		return fmt.Sprintf("%s is now %s", e.ActorID, e.Condition)
	}
	return fmt.Sprintf("%s is no longer %s", e.ActorID, e.Condition)
}

// AskIssuedEvent freezes the game and requests input from a target.
// Options contains the commands the target may choose from to resolve.
type AskIssuedEvent struct {
	TargetID string   `json:"target_id"`
	Options  []string `json:"options"`
}

func (e *AskIssuedEvent) Type() string { return "AskIssuedEvent" }
func (e *AskIssuedEvent) Apply(state *GameState) error {
	state.Metadata["pending_ask"] = map[string]any{
		"target":  e.TargetID,
		"options": e.Options,
	}
	return nil
}
func (e *AskIssuedEvent) Message() string {
	return fmt.Sprintf("Waiting for %s to respond", e.TargetID)
}

// HintEvent is a display-only message that is not persisted.
type HintEvent struct {
	MessageStr string `json:"message"`
}

func (e *HintEvent) Type() string                 { return "HintEvent" }
func (e *HintEvent) Apply(state *GameState) error { return nil }
func (e *HintEvent) Message() string              { return e.MessageStr }

// DiceRolledEvent records the result of a dice roll.
type DiceRolledEvent struct {
	ActorID string `json:"actor_id"`
	Dice    string `json:"dice"`
	Result  int    `json:"result"`
}

func (e *DiceRolledEvent) Type() string                 { return "DiceRolledEvent" }
func (e *DiceRolledEvent) Apply(state *GameState) error { return nil }
func (e *DiceRolledEvent) Message() string {
	return fmt.Sprintf("%s rolled %s = %d", e.ActorID, e.Dice, e.Result)
}

// MetadataChangedEvent stores or updates arbitrary data in global game metadata.
type MetadataChangedEvent struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func (e *MetadataChangedEvent) Type() string { return "MetadataChangedEvent" }
func (e *MetadataChangedEvent) Apply(state *GameState) error {
	state.Metadata[e.Key] = e.Value
	return nil
}
func (e *MetadataChangedEvent) Message() string {
	return fmt.Sprintf("metadata.%s updated", e.Key)
}

// CheckEvent records the boolean result of an ability check or skill contest.
type CheckEvent struct {
	ActorID string `json:"actor_id"`
	Check   string `json:"check"`
	Passed  bool   `json:"passed"`
}

func (e *CheckEvent) Type() string { return "CheckEvent" }
func (e *CheckEvent) Apply(state *GameState) error {
	state.Metadata["last_check"] = map[string]any{
		"actor":  e.ActorID,
		"check":  e.Check,
		"passed": e.Passed,
	}
	return nil
}
func (e *CheckEvent) Message() string {
	result := "failed"
	if e.Passed {
		result = "passed"
	}
	return fmt.Sprintf("%s %s check: %s", e.ActorID, e.Check, result)
}

// --- Helpers ---

// toInt safely extracts an int from various numeric types (int, int64, float64, string).
func toInt(val any) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i, true
		}
	}
	return 0, false
}
