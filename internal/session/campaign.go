package session

import (
	"fmt"
	"os"
	"path/filepath"
)

// CampaignManager bridges configuration settings with local file organization.
// It handles directory creation and path resolution for campaign data,
// independent of the event storage mechanism.
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

// GetLogPath returns the path to the event log file for a campaign.
func (c *CampaignManager) GetLogPath(world, campaign string) string {
	return filepath.Join(c.GetCampaignPath(world, campaign), "log.jsonl")
}

// Create generates standard directory structure for an initialized campaign.
// Returns the log file path (caller is responsible for opening it with the appropriate store).
func (c *CampaignManager) Create(world, campaign string) (string, error) {
	path := c.GetCampaignPath(world, campaign)

	dirs := []string{
		path,
		filepath.Join(path, "characters"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	logPath := c.GetLogPath(world, campaign)
	return logPath, nil
}

// Load verifies the campaign directory exists and returns the log file path.
// Returns the log file path (caller is responsible for opening it with the appropriate store).
func (c *CampaignManager) Load(world, campaign string) (string, error) {
	path := c.GetCampaignPath(world, campaign)
	if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("campaign target folder not properly found: %s", path)
	}

	logPath := c.GetLogPath(world, campaign)
	return logPath, nil
}
