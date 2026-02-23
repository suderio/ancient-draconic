package engine

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/suderio/ancient-draconic/internal/parser"
)

var mockDiceQueue []int

// MockDice prepares a sequence of deterministic results for the next calls to Roll
func MockDice(results []int) {
	mockDiceQueue = results
}

// ResetMockDice clears the deterministic queue
func ResetMockDice() {
	mockDiceQueue = nil
}

// RollResult contains the finalized answer alongside the raw rolls used
type RollResult struct {
	Total    int
	RawRolls []int
	Kept     []int
	Dropped  []int
	Modifier int
}

// safeRand fetches a strongly uniform random integer via crypto/rand
func safeRand(max int) int {
	if max <= 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64()) + 1 // Convert 0-(Max-1) to 1-Max
}

var diceRegex = regexp.MustCompile(`(?i)^(\d*)[d](\d+)(k[hl]\d+|[ad])?([+-]\d+)?$`)

// Roll processes an ast.DiceExpr into a mathematically randomized RollResult
func Roll(expr *parser.DiceExpr) (RollResult, error) {
	if expr == nil || expr.Raw == "" {
		return RollResult{}, fmt.Errorf("dice expression cannot be nil or empty")
	}

	res := RollResult{}

	// Normalize notation and parse chunks
	raw := strings.ReplaceAll(expr.Raw, " ", "")

	matches := diceRegex.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return res, fmt.Errorf("invalid dice expression format: %s", raw)
	}

	numStr, sidesStr, keepDropStr, modStr := matches[1], matches[2], matches[3], matches[4]

	// 1. Number of Dice
	numDice := 1
	if numStr != "" {
		numDice, _ = strconv.Atoi(numStr)
	}

	// 2. Sides
	sides, _ := strconv.Atoi(sidesStr)
	if sides <= 0 {
		return res, fmt.Errorf("cannot roll a die with 0 or negative sides")
	}

	// 3. Modifiers (Adv/Dis / Keep / Drop)
	keepTotal := numDice
	isHighest := true

	if keepDropStr != "" {
		kdLower := strings.ToLower(keepDropStr)
		if kdLower == "a" {
			numDice = 2
			keepTotal = 1
			isHighest = true
		} else if kdLower == "d" {
			numDice = 2
			keepTotal = 1
			isHighest = false
		} else if strings.HasPrefix(kdLower, "k") {
			// e.g. kh2 or kl1
			isHighest = strings.Contains(kdLower, "h")
			// Extract number
			kdCountStr := kdLower[2:]
			if kdCountStr != "" {
				parsed, err := strconv.Atoi(kdCountStr)
				if err == nil {
					keepTotal = parsed
				}
			}
		}
	}

	// 4. Generate Raw Rolls
	for i := 0; i < numDice; i++ {
		val := 0
		if len(mockDiceQueue) > 0 {
			val = mockDiceQueue[0]
			mockDiceQueue = mockDiceQueue[1:]
		} else {
			val = safeRand(sides)
		}
		res.RawRolls = append(res.RawRolls, val)
	}

	// 5. Resolve Keep/Drop Sorting
	// Clone array to safely sort without mutating the original order recording
	sorted := make([]int, len(res.RawRolls))
	copy(sorted, res.RawRolls)

	if keepTotal > numDice {
		keepTotal = numDice // Cannot keep more than rolled
	} else if keepTotal < 0 {
		keepTotal = 0
	}

	if isHighest {
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	} else {
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	}

	if keepTotal < numDice {
		res.Kept = sorted[:keepTotal]
		res.Dropped = sorted[keepTotal:]
	} else {
		res.Kept = sorted
	}

	// 6. Sum total + flat modifiers
	for _, val := range res.Kept {
		res.Total += val
	}

	if modStr != "" {
		modVal, err := strconv.Atoi(modStr)
		if err == nil {
			res.Modifier = modVal
			res.Total += modVal
		}
	}

	return res, nil
}
