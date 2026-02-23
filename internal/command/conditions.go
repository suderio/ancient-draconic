package command

import (
	"strings"

	"github.com/suderio/ancient-draconic/internal/engine"
)

// GetConditionMatrix returns three booleans for a checking actor:
// autoFail, hasAdvantage, hasDisadvantage based on standard D&D conditions rules.
func GetConditionMatrixForCheck(actorID string, checkType []string, state *engine.GameState) (bool, bool, bool) {
	ent, ok := state.Entities[actorID]
	if !ok {
		return false, false, false
	}

	autoFail := false
	hasAdv := false
	hasDis := false

	strCheck := strings.ToLower(strings.Join(checkType, " "))
	isDexSave := strings.Contains(strCheck, "dex") && strings.Contains(strCheck, "save")
	isStrSave := strings.Contains(strCheck, "str") && strings.Contains(strCheck, "save")

	for _, cond := range ent.Conditions {
		switch cond {
		case "blinded", "deafened":
			// Technically sight/hearing based, but we'll leave it to GM discretion or assume no autofail strictly here for generic.
		case "exhaustion":
			hasDis = true // level 1+
		case "frightened", "poisoned":
			if !strings.Contains(strCheck, "save") {
				hasDis = true // Disadvantage on ability checks
			}
		case "paralyzed", "petrified", "stunned", "unconscious":
			if isDexSave || isStrSave {
				autoFail = true
			}
		case "restrained":
			if isDexSave {
				hasDis = true
			}
		case "Dodging":
			if isDexSave && !IsIncapacitated(ent) {
				hasAdv = true
			}
		default:
			if strings.HasPrefix(cond, "HelpedCheck:") {
				hasAdv = true
			}
		}
	}

	return autoFail, hasAdv, hasDis
}

// GetConditionMatrixForAttack analyzes the attacker and target for Advantage/Disadvantage
func GetConditionMatrixForAttack(attackerID, targetID string, state *engine.GameState) (bool, bool) {
	attacker, aOk := state.Entities[attackerID]
	target, tOk := state.Entities[targetID]

	hasAdv := false
	hasDis := false

	if aOk {
		for _, cond := range attacker.Conditions {
			switch cond {
			case "blinded", "exhaustion", "frightened", "poisoned", "prone", "restrained":
				hasDis = true
			case "invisible":
				hasAdv = true
			}
		}
	}

	if tOk {
		for _, cond := range target.Conditions {
			switch cond {
			case "blinded", "paralyzed", "petrified", "restrained", "stunned", "unconscious":
				hasAdv = true
			case "invisible":
				hasDis = true
			case "prone":
				hasAdv = true // Assuming melee for now, or letting GM decide
			case "Dodging":
				if !IsIncapacitated(target) {
					hasDis = true
				}
			default:
				if strings.HasPrefix(cond, "HelpedAttack:") {
					hasAdv = true
				}
			}
		}
	}

	// Double positive cancels out
	if hasAdv && hasDis {
		return false, false
	}

	return hasAdv, hasDis
}

// IsIncapacitated checks if an entity is unable to take actions or reactions
func IsIncapacitated(ent *engine.Entity) bool {
	for _, c := range ent.Conditions {
		switch c {
		case "incapacitated", "paralyzed", "petrified", "stunned", "unconscious":
			return true
		}
	}
	return false
}
