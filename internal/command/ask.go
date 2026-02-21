package command

import (
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
	"strings"
)

// ExecuteAsk validates a GM-issued check request and freezes the encounter state
func ExecuteAsk(cmd *parser.AskCmd, state *engine.GameState) ([]engine.Event, error) {
	if err := ValidateGM(cmd.Actor); err != nil {
		return nil, err
	}

	var cleanTargets []string
	for _, t := range cmd.Targets {
		clean := strings.ToLower(t)
		cleanTargets = append(cleanTargets, clean)
	}

	evt := &engine.AskIssuedEvent{
		Targets: cleanTargets,
		Check:   cmd.Check,
		DC:      cmd.DC,
	}

	if cmd.Fails != nil {
		evt.Fails = &engine.RollConsequence{
			IsDamage:   cmd.Fails.IsDamage != "",
			HalfDamage: cmd.Fails.HalfDamage,
			Condition:  cmd.Fails.Condition,
		}
		if cmd.Fails.DamageDice != nil {
			evt.Fails.DamageDice = cmd.Fails.DamageDice.Raw
		}
	}

	if cmd.Succeeds != nil {
		evt.Succeeds = &engine.RollConsequence{
			IsDamage:   cmd.Succeeds.IsDamage != "",
			HalfDamage: cmd.Succeeds.HalfDamage,
			Condition:  cmd.Succeeds.Condition,
		}
		if cmd.Succeeds.DamageDice != nil {
			evt.Succeeds.DamageDice = cmd.Succeeds.DamageDice.Raw
		}
	}

	return []engine.Event{evt}, nil
}
