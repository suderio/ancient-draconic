//go:build ignore

package main

import (
	"fmt"

	"github.com/suderio/ancient-draconic/internal/engine"
)

func main() {
	eval, _ := engine.NewLuaEvaluator(nil)
	m, err := eval.LoadManifestLua("world/dnd5e/manifest.lua")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Loaded %d commands\n", len(m.Commands))
	for k, v := range m.Commands {
		fmt.Printf("Key: %q, Name: %q, %d Game steps\n", k, v.Name, len(v.Game))
	}
}
