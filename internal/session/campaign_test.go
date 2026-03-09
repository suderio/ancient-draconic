package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCampaignManager_GetPaths(t *testing.T) {
	m := NewCampaignManager("/tmp/worlds")
	assert.Equal(t, "/tmp/worlds/dnd5e/mycampaign", m.GetCampaignPath("dnd5e", "mycampaign"))
	assert.Equal(t, "/tmp/worlds/dnd5e/mycampaign/log.jsonl", m.GetLogPath("dnd5e", "mycampaign"))
}

func TestCampaignManager_Create(t *testing.T) {
	dir := t.TempDir()
	m := NewCampaignManager(dir)

	logPath, err := m.Create("dnd5e", "testcampaign")
	assert.NoError(t, err)
	assert.Contains(t, logPath, "log.jsonl")
}

func TestCampaignManager_Load_NotFound(t *testing.T) {
	m := NewCampaignManager("/nonexistent")
	_, err := m.Load("dnd5e", "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not properly found")
}

func TestCampaignManager_Load_Success(t *testing.T) {
	dir := t.TempDir()
	m := NewCampaignManager(dir)

	// Create first, then load
	_, err := m.Create("dnd5e", "testcampaign")
	assert.NoError(t, err)

	logPath, err := m.Load("dnd5e", "testcampaign")
	assert.NoError(t, err)
	assert.Contains(t, logPath, "log.jsonl")
}
