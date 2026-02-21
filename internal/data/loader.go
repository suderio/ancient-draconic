package data

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed srd/**/*.yaml
var srdFS embed.FS

// Loader handles reading and instantiating records from the read-only data layer
type Loader struct {
	dataDirs []string
}

// NewLoader initializes a new Data Loader with the given data directory fallback hierarchy
func NewLoader(dataDirs []string) *Loader {
	return &Loader{
		dataDirs: dataDirs,
	}
}

// LoadCharacter constructs a typed Character object by searching through the data directories sequentially
func (l *Loader) LoadCharacter(name string) (*Character, error) {
	var c Character
	ref := filepath.Join("characters", fmt.Sprintf("%s.yaml", name))
	if err := l.load(ref, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadMonster constructs a typed Monster object by searching through the data directories sequentially
func (l *Loader) LoadMonster(name string) (*Monster, error) {
	var m Monster
	dashName := strings.ReplaceAll(strings.ToLower(name), " ", "-")
	ref := filepath.Join("monsters", fmt.Sprintf("%s.yaml", dashName))
	if err := l.load(ref, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (l *Loader) load(ref string, target interface{}) error {
	// 1. Check external directories (Campaign/World/etc)
	for _, dir := range l.dataDirs {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, ref)
		f, err := os.Open(path)
		if err == nil {
			defer f.Close()
			decoder := yaml.NewDecoder(f)
			if err := decoder.Decode(target); err != nil {
				return fmt.Errorf("failed to decode yaml reference %s: %w", ref, err)
			}
			return nil
		}
	}

	// 2. Check Embedded FS (Baseline SRD)
	// We handle the con.yaml -> constitution.yaml mapping for Windows portability in the binary
	internalRef := ref
	if ref == filepath.Join("ability-scores", "con.yaml") || ref == "ability-scores/con.yaml" {
		internalRef = filepath.Join("ability-scores", "constitution.yaml")
	}

	// embed.FS uses forward slashes regardless of OS
	embeddedPath := "srd/" + filepath.ToSlash(internalRef)
	f, err := srdFS.Open(embeddedPath)
	if err == nil {
		defer f.Close()
		decoder := yaml.NewDecoder(f)
		if err := decoder.Decode(target); err != nil {
			return fmt.Errorf("failed to decode embedded yaml reference %s: %w", ref, err)
		}
		return nil
	}

	return fmt.Errorf("could not find or open reference %s in any available data directory or embedded storage", ref)
}
