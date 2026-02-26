package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInput_SimpleCommand(t *testing.T) {
	p := ParseInput("help")
	assert.Equal(t, "help", p.Command)
	assert.Equal(t, "GM", p.ActorID) // defaults to GM
}

func TestParseInput_CommandWithDice(t *testing.T) {
	p := ParseInput("roll dice: 2d6")
	assert.Equal(t, "roll", p.Command)
	assert.Equal(t, "2d6", p.Params["dice"])
}

func TestParseInput_CommandWithActor(t *testing.T) {
	p := ParseInput("attack by: Fighter to: Goblin")
	assert.Equal(t, "attack", p.Command)
	assert.Equal(t, "Fighter", p.ActorID)
	assert.Equal(t, []string{"Goblin"}, p.Targets)
}

func TestParseInput_MultiWordCommand(t *testing.T) {
	p := ParseInput("encounter start")
	assert.Equal(t, "encounter_start", p.Command)
}

func TestParseInput_MultiWordCommandWithParams(t *testing.T) {
	p := ParseInput("encounter start with: Fighter and Goblin")
	assert.Equal(t, "encounter_start", p.Command)
	assert.Equal(t, []string{"Fighter", "Goblin"}, p.Params["with"])
}

func TestParseInput_GrappleCommand(t *testing.T) {
	p := ParseInput("grapple by: Fighter to: Goblin")
	assert.Equal(t, "grapple", p.Command)
	assert.Equal(t, "Fighter", p.ActorID)
	assert.Equal(t, []string{"Goblin"}, p.Targets)
}

func TestParseInput_EmptyInput(t *testing.T) {
	p := ParseInput("")
	assert.Equal(t, "", p.Command)
}

func TestParseInput_MultipleParams(t *testing.T) {
	p := ParseInput("check by: Fighter skill: athletics dc: 15")
	assert.Equal(t, "check", p.Command)
	assert.Equal(t, "Fighter", p.ActorID)
	assert.Equal(t, "athletics", p.Params["skill"])
	assert.Equal(t, "15", p.Params["dc"])
}
