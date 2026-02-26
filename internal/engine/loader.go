package engine

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadManifest reads and parses a manifest YAML file into a Manifest struct.
func LoadManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest %s: %w", path, err)
	}
	defer f.Close()

	var m Manifest
	if err := yaml.NewDecoder(f).Decode(&m); err != nil {
		return nil, fmt.Errorf("failed to decode manifest %s: %w", path, err)
	}

	// Initialize Commands map if nil (empty manifest)
	if m.Commands == nil {
		m.Commands = make(map[string]CommandDef)
	}

	return &m, nil
}

// LoadEntity reads and parses a character or monster YAML file into an Entity struct.
// After loading, all nil maps are initialized to empty maps to prevent nil-map panics.
func LoadEntity(path string) (*Entity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open entity %s: %w", path, err)
	}
	defer f.Close()

	var e Entity
	if err := yaml.NewDecoder(f).Decode(&e); err != nil {
		return nil, fmt.Errorf("failed to decode entity %s: %w", path, err)
	}

	// Ensure all maps are initialized
	if e.Types == nil {
		e.Types = make([]string, 0)
	}
	if e.Classes == nil {
		e.Classes = make(map[string]string)
	}
	if e.Stats == nil {
		e.Stats = make(map[string]int)
	}
	if e.Resources == nil {
		e.Resources = make(map[string]int)
	}
	if e.Spent == nil {
		e.Spent = make(map[string]int)
	}
	if e.Conditions == nil {
		e.Conditions = make([]string, 0)
	}
	if e.Proficiencies == nil {
		e.Proficiencies = make(map[string]int)
	}
	if e.Statuses == nil {
		e.Statuses = make(map[string]string)
	}
	if e.Inventory == nil {
		e.Inventory = make(map[string]int)
	}

	return &e, nil
}
