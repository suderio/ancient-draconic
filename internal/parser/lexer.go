package parser

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Lexer maps the raw string tokens out for our AST definitions.
// Basic whitespace elision is enough for our grammar.
var Lexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Keyword", Pattern: `(?i)\b(?:roll|encounter|start|end|with|and|add|initiative|by|attack|damage|turn|to|dice|hint|ask|check|of|dc|fails|succeeds|half)\b`},
	{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
	{Name: "DiceMacro", Pattern: `\d+[dD]\d+(?:[kK][hHlL]?\d+|[aAdD])?(?:[+-]\d+)?`},
	{Name: "Int", Pattern: `[0-9]+`},
	{Name: "Punct", Pattern: `[:]`},
	{Name: "Whitespace", Pattern: `[ \t]+`},
})

// Build creates our parser based on the struct tags in `ast.go`
func Build() *participle.Parser[Command] {
	return participle.MustBuild[Command](
		participle.Lexer(Lexer),
		participle.Elide("Whitespace"),
	)
}
