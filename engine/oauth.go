package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitHub OAuth App credentials for DevBase
// You'll need to create a GitHub OAuth App at https://github.com/settings/developers
const (
	// Note: This is a placeholder. For production use, create your own OAuth App
	// at https://github.com/settings/developers and replace this with your client ID
	ClientID = "Ov23liNemMNmQpa1yLxG" // Placeholder - replace with actual OAuth App client ID
)

// DeviceCodeResponse represents the response from GitHub's device code endpoint
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse represents the response from GitHub's access token endpoint
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// OAuthClient handles GitHub OAuth device flow operations
type OAuthClient struct {
	ClientID string
}

// NewOAuthClient creates a new OAuthClient
func NewOAuthClient() *OAuthClient {
	return &OAuthClient{
		ClientID: ClientID,
	}
}

// InitiateDeviceFlow starts the OAuth device flow
func (c *OAuthClient) InitiateDeviceFlow() (*DeviceCodeResponse, error) {
	// Check if we have a valid client ID
	if c.ClientID == "" || c.ClientID == "Iv1.0000000000000000" {
		return nil, fmt.Errorf("OAuth not configured: Please create a GitHub OAuth App at https://github.com/settings/developers and update the ClientID constant")
	}

	url := "https://github.com/login/device/code"

	data := map[string]string{
		"client_id": c.ClientID,
		"scope":     "gist repo",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("OAuth client not found: Please ensure your GitHub OAuth App client ID is correct")
		}
		if resp.StatusCode == 422 {
			return nil, fmt.Errorf("OAuth configuration error: Please check your GitHub OAuth App settings")
		}
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &deviceResp, nil
}

// PollForAccessToken polls GitHub for the access token
func (c *OAuthClient) PollForAccessToken(deviceCode string, interval int) (string, error) {
	url := "https://github.com/login/oauth/access_token"

	pollInterval := time.Duration(interval) * time.Second
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Timeout after 10 minutes
	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("authentication timeout: user did not authorize within 10 minutes")

		case <-ticker.C:
			data := map[string]string{
				"client_id":   c.ClientID,
				"device_code": deviceCode,
				"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				return "", fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			if err != nil {
				return "", fmt.Errorf("failed to create request: %w", err)
			}

			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("failed to execute request: %w", err)
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return "", fmt.Errorf("failed to read response: %w", err)
			}

			var tokenResp AccessTokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				return "", fmt.Errorf("failed to unmarshal response: %w", err)
			}

			// Check for errors
			if tokenResp.Error != "" {
				switch tokenResp.Error {
				case "authorization_pending":
					// User hasn't authorized yet, continue polling
					continue
				case "slow_down":
					// Slow down polling
					ticker.Reset(pollInterval + 5*time.Second)
					continue
				case "expired_token":
					return "", fmt.Errorf("device code expired: user took too long to authorize")
				case "access_denied":
					return "", fmt.Errorf("access denied: user cancelled authorization")
				default:
					return "", fmt.Errorf("OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
				}
			}

			// Success! We have an access token
			if tokenResp.AccessToken != "" {
				return tokenResp.AccessToken, nil
			}
		}
	}
}

// ValidateToken checks if the GitHub token is valid by making a test API call
func (c *OAuthClient) ValidateToken(token string) error {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
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

// GitHubRepository represents a GitHub repository from the API
type GitHubRepository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	CloneURL    string `json:"clone_url"`
	HTMLURL     string `json:"html_url"`
	Private     bool   `json:"private"`
	Language    string `json:"language"`
	UpdatedAt   string `json:"updated_at"`
}

// FetchUserRepositories retrieves all repositories for the authenticated user
func (c *OAuthClient) FetchUserRepositories(token string) ([]GitHubRepository, error) {
	var allRepos []GitHubRepository
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("https://api.github.com/user/repos?per_page=%d&page=%d&sort=updated&visibility=all&affiliation=owner,collaborator,organization_member", perPage, page)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repositories: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
		}

		var repos []GitHubRepository
		if err := json.Unmarshal(body, &repos); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)

		// If we got fewer than perPage results, we're done
		if len(repos) < perPage {
			break
		}

		page++
	}

	return allRepos, nil
}
