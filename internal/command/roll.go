package command

import (
	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/parser"
)

func ExecuteRoll(roll *parser.RollCmd) (engine.Event, error) {
	name := "System"
	if roll.Actor != nil {
		name = roll.Actor.Name
	}

	res, err := engine.Roll(roll.Dice)
	if err != nil {
		return nil, err
	}

	evt := &engine.DiceRolledEvent{
		ActorName: name,
		Total:     res.Total,
		RawRolls:  res.RawRolls,
		Kept:      res.Kept,
		Dropped:   res.Dropped,
		Modifier:  res.Modifier,
	}

	return evt, nil
}
