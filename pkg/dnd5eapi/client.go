package dnd5eapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const BaseURL = "https://www.dnd5eapi.co"

type Client struct {
	client  *http.Client
	dataDir string
	force   bool
}

func NewClient(dataDir string, force bool) *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		dataDir: dataDir,
		force:   force,
	}
}

type APIListResponse struct {
	Count   int `json:"count"`
	Results []struct {
		Index string `json:"index"`
		Name  string `json:"name"`
		URL   string `json:"url"`
	} `json:"results"`
}

func (c *Client) FetchList(endpoint string) (*APIListResponse, error) {
	url := fmt.Sprintf("%s/api/2014/%s", BaseURL, endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: %s", url, resp.Status)
	}

	var list APIListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	return &list, nil
}

func (c *Client) FetchItem(url string) (map[string]interface{}, error) {
	fullURL := fmt.Sprintf("%s%s", BaseURL, url)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: %s", fullURL, resp.Status)
	}

	var item map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}

	return item, nil
}

func (c *Client) DownloadImage(imageURL string) (string, error) {
	fullURL := fmt.Sprintf("%s%s", BaseURL, imageURL)

	// Create local path based on the URL path
	// Example: /api/images/monsters/aboleth.png -> <dataDir>/images/monsters/aboleth.png
	// Actually, the requirements say "Construct local file path (data/magic-items/weapon.png)"
	// Let's strip "/api/images/" and place it in the respective category dir? Or keep it in its path structure?
	// The url is typically /api/images/monsters/aboleth.png or similar.
	// Let's just strip "/api/images/" or use the path directly.
	// We'll strip "/api/" and then we get "images/monsters/aboleth.png". We can save it as <dataDir>/images/monsters/... or just keep the full path.
	// Let's use the URL path to define the local path relative to dataDir, stripping "/api/" so it goes to <dataDir>/images/...
	localRelativePath := strings.TrimPrefix(imageURL, "/api/")
	localPath := filepath.Join(c.dataDir, localRelativePath)

	if !c.force {
		if _, err := os.Stat(localPath); err == nil {
			return localRelativePath, nil // Image exists, skip
		}
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image %s: %s", fullURL, resp.Status)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	return localRelativePath, nil
}

func (c *Client) Transform(data interface{}, currentEndpoint string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		transformed := make(map[string]interface{})
		for key, val := range v {
			if key == "url" {
				if strVal, ok := val.(string); ok && strings.HasPrefix(strVal, "/api/2014/") {
					transformed["ref"] = strings.TrimPrefix(strVal, "/api/2014/") + ".yaml"
					continue
				}
			}
			if key == "resource_list_url" {
				if strVal, ok := val.(string); ok && strings.HasPrefix(strVal, "/api/2014/") {
					transformed["resource_list_ref"] = strings.TrimPrefix(strVal, "/api/2014/") + ".yaml"
					continue
				}
			}
			// Special properties
			if key == "subclass_levels" || key == "class_levels" || key == "spell" || key == "feature" || key == "reference" {
				if strVal, ok := val.(string); ok && strings.HasPrefix(strVal, "/api/2014/") {
					relPath := strings.TrimPrefix(strVal, "/api/2014/") + ".yaml"
					transformed[key] = relPath
					// We also need to trigger a download for subclass_levels and class_levels since they are separate API calls
					// We'll handle this download outside this pure transform step or just do it inline here since we have the client.
					if key == "subclass_levels" || key == "class_levels" {
						c.downloadExtraLevel(strVal, relPath)
					}
					continue
				}
			}
			if key == "image" {
				if strVal, ok := val.(string); ok {
					localPath, err := c.DownloadImage(strVal)
					if err == nil {
						transformed[key] = localPath
					} else {
						transformed[key] = strVal
					}
					continue
				}
			}
			transformed[key] = c.Transform(val, currentEndpoint)
		}
		return transformed
	case []interface{}:
		var transformed []interface{}
		for _, item := range v {
			transformed = append(transformed, c.Transform(item, currentEndpoint))
		}
		return transformed
	default:
		return v
	}
}

func (c *Client) downloadExtraLevel(url string, relativeTarget string) {
	localPath := filepath.Join(c.dataDir, relativeTarget)
	if !c.force {
		if _, err := os.Stat(localPath); err == nil {
			return
		}
	}

	// Throttle slightly
	time.Sleep(100 * time.Millisecond)

	// We need to fetch it generically since levels could be a JSON array.
	fullURL := fmt.Sprintf("%s%s", BaseURL, url)
	resp, err := c.client.Get(fullURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var raw interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return
	}

	transformed := c.Transform(raw, "")

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return
	}

	f, err := os.Create(localPath)
	if err != nil {
		return
	}
	defer f.Close()

	yaml.NewEncoder(f).Encode(transformed)
}

func (c *Client) SaveItem(endpoint string, index string, data interface{}) error {
	relPath := fmt.Sprintf("%s/%s.yaml", endpoint, index)
	localPath := filepath.Join(c.dataDir, relPath)

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}
