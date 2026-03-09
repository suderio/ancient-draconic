package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManifest_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	err := os.WriteFile(path, []byte(`
commands:
  attack:
    name: attack
    help: "Perform an attack"
`), 0644)
	require.NoError(t, err)

	m, err := LoadManifest(path)
	require.NoError(t, err)
	assert.NotNil(t, m)
	assert.Contains(t, m.Commands, "attack")
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	_, err := LoadManifest("/nonexistent/manifest.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}

func TestLoadManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	err := os.WriteFile(path, []byte(":::invalid}}yaml"), 0644)
	require.NoError(t, err)

	_, err = LoadManifest(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestLoadManifest_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	err := os.WriteFile(path, []byte(""), 0644)
	require.NoError(t, err)

	// Empty YAML file returns EOF error
	_, err = LoadManifest(path)
	assert.Error(t, err)
}

func TestLoadEntity_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fighter.yaml")
	err := os.WriteFile(path, []byte(`
id: fighter
name: Fighter
types:
  - humanoid
stats:
  str: 18
  dex: 14
resources:
  hp: 45
`), 0644)
	require.NoError(t, err)

	e, err := LoadEntity(path)
	require.NoError(t, err)
	assert.Equal(t, "fighter", e.ID)
	assert.Equal(t, 18, e.Stats["str"])
	assert.NotNil(t, e.Conditions)
	assert.NotNil(t, e.Spent)
	assert.NotNil(t, e.Proficiencies)
	assert.NotNil(t, e.Statuses)
	assert.NotNil(t, e.Inventory)
	assert.NotNil(t, e.Classes)
}

func TestLoadEntity_FileNotFound(t *testing.T) {
	_, err := LoadEntity("/nonexistent/fighter.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}

func TestLoadEntity_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	err := os.WriteFile(path, []byte(":::invalid}}yaml"), 0644)
	require.NoError(t, err)

	_, err = LoadEntity(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}
