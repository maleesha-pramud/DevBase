package engine

import (
	"bytes"
	"devbase/db"
	"devbase/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GistClient handles GitHub Gist operations
type GistClient struct {
	Token  string // GitHub token
	GistID string // ID of the gist, empty if not created yet
}

// NewGistClient creates a new GistClient with token and loads existing gist ID from database
func NewGistClient(token string) (*GistClient, error) {
	gc := &GistClient{Token: token}

	// Load existing gist ID from database
	gistID, err := db.GetConfig("gist_id")
	if err == nil && gistID != "" {
		gc.GistID = gistID
	}

	return gc, nil
}

// ValidateToken checks if the GitHub token is valid by making a test API call
func (c *GistClient) ValidateToken() error {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	// Support both OAuth tokens (Bearer) and PATs (token)
	// Try Bearer first (OAuth), then fall back to token (PAT)
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid GitHub token")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API error during token validation: %d", resp.StatusCode)
	}

	return nil
}

// getAuthHeader returns the appropriate Authorization header value
func (c *GistClient) getAuthHeader() string {
	// OAuth tokens use Bearer, PATs use token
	// We'll default to Bearer for new OAuth flow
	return "Bearer " + c.Token
}

// SaveToGist saves project data to a GitHub Gist
func (c *GistClient) SaveToGist(projects []models.Project) error {
	// Prepare data for gist
	data := map[string]interface{}{
		"description": "DevBase project data backup",
		"public":      false,
		"files": map[string]interface{}{
			"devbase_projects.json": map[string]interface{}{
				"content": c.projectsToJSON(projects),
			},
		},
	}

	// If gistID is provided, update existing gist
	var url string
	var method string
	if c.GistID != "" {
		url = fmt.Sprintf("https://api.github.com/gists/%s", c.GistID)
		method = "PATCH"
	} else {
		url = "https://api.github.com/gists"
		method = "POST"
	}

	// Create request
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.getAuthHeader())
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API error: %s", string(body))
	}

	// Parse response to get gist ID (only for new gists)
	if c.GistID == "" {
		var gistResp struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &gistResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Store the new gist ID
		c.GistID = gistResp.ID
		if err := db.SetConfig("gist_id", c.GistID); err != nil {
			return fmt.Errorf("failed to save gist ID: %w", err)
		}
	}

	return nil
}

// LoadFromGist loads project data from a GitHub Gist
func (c *GistClient) LoadFromGist() ([]models.Project, error) {
	if c.GistID == "" {
		return nil, fmt.Errorf("no cloud backup found. Please sync to cloud first")
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", c.GistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.getAuthHeader())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		// Gist was deleted, clear the stored ID
		c.GistID = ""
		db.SetConfig("gist_id", "")
		return nil, fmt.Errorf("cloud backup not found (gist may have been deleted). Please sync to cloud first")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	// Parse gist response
	var gistResp struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}

	if err := json.Unmarshal(body, &gistResp); err != nil {
		return nil, fmt.Errorf("failed to parse gist response: %w", err)
	}

	// Extract project data from the gist file
	file, exists := gistResp.Files["devbase_projects.json"]
	if !exists {
		return nil, fmt.Errorf("devbase_projects.json not found in gist")
	}

	return c.jsonToProjects(file.Content)
}

// projectsToJSON converts projects slice to JSON string
func (c *GistClient) projectsToJSON(projects []models.Project) string {
	data, _ := json.MarshalIndent(projects, "", "  ")
	return string(data)
}

// jsonToProjects converts JSON string to projects slice
func (c *GistClient) jsonToProjects(jsonStr string) ([]models.Project, error) {
	var projects []models.Project
	if err := json.Unmarshal([]byte(jsonStr), &projects); err != nil {
		return nil, fmt.Errorf("failed to parse projects JSON: %w", err)
	}
	return projects, nil
}

// ListProjectsFromGist lists project names from a GitHub Gist without loading full data
func (c *GistClient) ListProjectsFromGist() ([]models.Project, error) {
	return c.LoadFromGist()
}
