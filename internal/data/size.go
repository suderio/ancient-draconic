package data

import "strings"

// Size represents the standard D&D 5e size categories.
type Size int

const (
	SizeTiny Size = iota
	SizeSmall
	SizeMedium
	SizeLarge
	SizeHuge
	SizeGargantuan
	SizeUnknown
)

var sizeMap = map[string]Size{
	"tiny":       SizeTiny,
	"small":      SizeSmall,
	"medium":     SizeMedium,
	"large":      SizeLarge,
	"huge":       SizeHuge,
	"gargantuan": SizeGargantuan,
}

// ParseSize converts a string into a comparable Size value.
func ParseSize(s string) Size {
	if val, ok := sizeMap[strings.ToLower(s)]; ok {
		return val
	}
	return SizeUnknown
}

// CanShove returns true if the attacker can shove the target based on size (at most one size larger).
func CanShove(attackerSize, targetSize Size) bool {
	if attackerSize == SizeUnknown || targetSize == SizeUnknown {
		return false
	}
	return int(targetSize) <= int(attackerSize)+1
}
