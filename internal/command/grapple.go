package command

import (
	"fmt"
	"strings"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/engine"
	"github.com/suderio/ancient-draconic/internal/parser"
	"github.com/suderio/ancient-draconic/internal/rules"
)

// ExecuteGrapple initiates a grapple attempt which requires GM adjudication
func ExecuteGrapple(cmd *parser.GrappleCmd, state *engine.GameState, loader *data.Loader, reg *rules.Registry) ([]engine.Event, error) {
	if state.IsFrozen() {
		return nil, engine.ErrSilentIgnore
	}

	actorName := "GM"
	if cmd.Actor != nil {
		actorName = cmd.Actor.Name
	}

	if len(state.TurnOrder) == 0 || state.CurrentTurn < 0 {
		return nil, fmt.Errorf("combat has not started")
	}

	currentActor := state.TurnOrder[state.CurrentTurn]
	// Basic turn order enforcement: only the active actor or GM can initiate
	if !strings.EqualFold(actorName, "GM") && !strings.EqualFold(actorName, currentActor) {
		return nil, engine.ErrSilentIgnore
	}

	ent, ok := state.Entities[currentActor]
	if !ok {
		return nil, fmt.Errorf("actor %s not found in encounter", currentActor)
	}

	if ent.ActionsRemaining <= 0 {
		return nil, fmt.Errorf("%s has no actions remaining this turn", currentActor)
	}

	// For grapple, we automatically trigger adjudication as per user requirements.
	// If we haven't asked for adjudication yet, do it now.
	if state.PendingAdjudication == nil {
		originalCmd := fmt.Sprintf("grapple by: %s to: %s", currentActor, cmd.Target)
		return []engine.Event{
			&engine.AdjudicationStartedEvent{
				OriginalCommand: originalCmd,
			},
		}, nil
	}

	// Double check freeze if not approved
	if !state.PendingAdjudication.Approved {
		return nil, fmt.Errorf("action is still pending GM adjudication")
	}

	// If we are here, it means adjudication was just cleared (Allowed)
	// Now we proceed with the grapple contest: Target makes a STR or DEX save.
	// For now, we'll ask for a generic "Strength or Dexterity save"
	// and let the internal logic handle the "choose one" better later if needed.

	// DC calculation (2024 revision): 8 + Strength modifier + Proficiency bonus
	dc := 10 // Baseline fallback
	if ent, ok := state.Entities[currentActor]; ok {
		// Try to load as character
		char, err := loader.LoadCharacter(ent.ID)
		if err == nil {
			dc = 8 + data.CalculateModifier(char.Strength) + char.ProficiencyBonus
		} else {
			// Try to load as monster
			mon, err := loader.LoadMonster(ent.ID)
			if err == nil {
				dc = 8 + data.CalculateModifier(mon.Strength) + mon.ProficiencyBonus
			}
		}
	}

	return []engine.Event{
		&engine.GrappleTakenEvent{
			Attacker: currentActor,
			Target:   cmd.Target,
		},
		&engine.AskIssuedEvent{
			Targets: []string{cmd.Target},
			Check:   []string{"strength", "save", "or", "dexterity", "save"},
			DC:      dc,
			Fails: &engine.RollConsequence{
				Condition: fmt.Sprintf("grappledby:%s", currentActor),
			},
		},
	}, nil
}
