package command

// Adjudication commands allow the GM to authorize or reject complex actions
// that require manual intervention or interpretation of rules.

import (
	"fmt"

	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

// ExecuteAdjudicate starts a GM authorization flow.
// It freezes the system until the GM issues an 'allow' or 'deny' command.
func ExecuteAdjudicate(cmd *parser.AdjudicateCmd) ([]engine.Event, error) {
	return []engine.Event{
		&engine.AdjudicationStartedEvent{
			OriginalCommand: cmd.Command,
		},
	}, nil
}

// ExecuteAllow fulfills a pending adjudication successfully.
// The session will then re-execute the original command that was pending.
func ExecuteAllow(cmd *parser.AllowCmd, state *engine.GameState) ([]engine.Event, error) {
	if cmd.Actor == nil {
		cmd.Actor = &parser.ActorExpr{Name: "GM"}
	}
	if err := ValidateGM(cmd.Actor); err != nil {
		return nil, err
	}
	if state.PendingAdjudication == nil {
		return nil, fmt.Errorf("conflict: no pending action to allow")
	}
	return []engine.Event{
		&engine.AdjudicationResolvedEvent{Allowed: true},
	}, nil
}

// ExecuteDeny rejects a pending adjudication
func ExecuteDeny(cmd *parser.DenyCmd, state *engine.GameState) ([]engine.Event, error) {
	if cmd.Actor == nil {
		cmd.Actor = &parser.ActorExpr{Name: "GM"}
	}
	if err := ValidateGM(cmd.Actor); err != nil {
		return nil, err
	}
	if state.PendingAdjudication == nil {
		return nil, fmt.Errorf("conflict: no pending action to deny")
	}
	return []engine.Event{
		&engine.AdjudicationResolvedEvent{Allowed: false},
	}, nil
}
