package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	for _, dir := range l.dataDirs {
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
	return fmt.Errorf("could not find or open reference %s in any available data directory", ref)
}
