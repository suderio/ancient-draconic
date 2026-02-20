package persistence

import (
	"fmt"
	"os"
	"path/filepath"
)

// CampaignManager bridges configuration settings with local file organization.
type CampaignManager struct {
	WorldsDir string
}

// NewCampaignManager returns manager localized to the specified workspace setting directory.
func NewCampaignManager(worldsDir string) *CampaignManager {
	return &CampaignManager{WorldsDir: worldsDir}
}

// GetCampaignPath produces safe joined absolute dir paths.
func (c *CampaignManager) GetCampaignPath(world, campaign string) string {
	return filepath.Join(c.WorldsDir, world, campaign)
}

// Create generates standard structure for an initialized world tracker session.
func (c *CampaignManager) Create(world, campaign string) (*Store, error) {
	path := c.GetCampaignPath(world, campaign)

	dirs := []string{
		path,
		filepath.Join(path, "characters"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	logPath := filepath.Join(path, "log.jsonl")
	return NewStore(logPath)
}

// Load verifies and grabs path from active target sessions.
func (c *CampaignManager) Load(world, campaign string) (*Store, error) {
	path := c.GetCampaignPath(world, campaign)
	if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("campaign target folder not properly found: %s", path)
	}

	logPath := filepath.Join(path, "log.jsonl")
	return NewStore(logPath)
}
