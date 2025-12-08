package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"devbase/db"
	"devbase/engine"
	"devbase/models"
)

// Custom message types for optimistic UI updates

// ArchiveMsg is sent when an archive operation completes
type ArchiveMsg struct {
	projectID uint
	err       error
	// Store original item for rollback on failure
	originalItem projectItem
	originalIdx  int
}

// RestoreMsg is sent when a restore operation completes
type RestoreMsg struct {
	projectID uint
	err       error
	// Store original item for rollback on failure
	originalItem projectItem
	originalIdx  int
}

// ErrorMsg displays an error message to the user
type ErrorMsg struct {
	err error
}

// OpenProjectMsg is sent when opening a project in VS Code completes
type OpenProjectMsg struct {
	projectID uint
	err       error
}

// OpenBrowserMsg is sent when opening a URL in the browser completes
type OpenBrowserMsg struct {
	url string
	err error
}

// RunProjectMsg is sent when running a project completes
type RunProjectMsg struct {
	projectPath string
	err         error
}

// ScanCompleteMsg is sent when directory scan completes
type ScanCompleteMsg struct {
	projectsFound   int
	projectsAdded   int
	projectsRemoved int
	err             error
}

// ClearAllMsg is sent when clearing all projects completes
type ClearAllMsg struct {
	count int
	err   error
}

// SyncToCloudMsg is sent when syncing projects to cloud completes
type SyncToCloudMsg struct {
	gistID string
	err    error
}

// LoadFromCloudMsg is sent when loading projects from cloud completes
type LoadFromCloudMsg struct {
	projectsLoaded int
	err            error
}

// ListCloudProjectsMsg is sent when listing projects from cloud completes
type ListCloudProjectsMsg struct {
	projects []models.Project
	err      error
}

// LoadSelectedProjectsMsg is sent when loading selected projects from cloud completes
type LoadSelectedProjectsMsg struct {
	projectsLoaded int
	err            error
}

// OAuthDeviceCodeMsg is sent when device code is obtained from GitHub
type OAuthDeviceCodeMsg struct {
	deviceCode      string
	userCode        string
	verificationURI string
	interval        int
	err             error
}

// OAuthCompleteMsg is sent when OAuth authentication completes
type OAuthCompleteMsg struct {
	accessToken string
	err         error
}

// projectItem wraps a Project and implements the list.Item interface
type projectItem struct {
	project   models.Project
	isLoading bool // Track if operation is in progress
}

// FilterValue implements list.Item
func (i projectItem) FilterValue() string {
	return i.project.Name
}

// Title implements list.DefaultItem
func (i projectItem) Title() string {
	title := i.project.Name

	// Add GitHub indicator
	if i.project.RepoURL != "" {
		title = "🔗 " + title
	}

	if i.isLoading {
		return title + " [Processing...]"
	}
	if i.project.Status == "archived" {
		return title + " [Archived]"
	}
	return title
}

// Description implements list.DefaultItem
func (i projectItem) Description() string {
	desc := ""
	if i.project.Path != "" {
		desc = i.project.Path
	} else {
		desc = i.project.Status
	}

	// Add repo URL info if available
	if i.project.RepoURL != "" {
		desc += " • " + i.project.RepoURL
	}

	return desc
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF0000")).
	Bold(true)

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#00FFFF")).
	Bold(true)

var subtitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#888888"))

// screenState represents the current screen being displayed
type screenState int

const (
	screenSetupPath screenState = iota
	screenSetupGitHub
	screenSetupToken
	screenOAuthWaiting
	screenCloudSelect
	screenList
)

// CloneMsg is sent when a clone operation completes
type CloneMsg struct {
	projectName string
	projectPath string
	err         error
}

// model represents the Bubble Tea application model
type model struct {
	screen               screenState
	pathInput            textinput.Model
	tokenInput           textinput.Model
	list                 list.Model
	errorMessage         string
	statusMessage        string
	isScanning           bool
	confirmClearAll      bool
	confirmArchive       bool
	archiveConfirmInput  textinput.Model
	archiveProject       *projectItem
	archiveIdx           int
	confirmClone         bool
	cloneInput           textinput.Model
	cloudProjects        []models.Project
	selectedCloudIndices []int
	cloudCursorIndex     int
	cloudFilterInput     textinput.Model
	cloudFiltering       bool
	rootScanPath         string
	width                int
	height               int
	ready                bool
	// OAuth flow fields
	oauthDeviceCode      string
	oauthUserCode        string
	oauthVerificationURI string
	oauthInterval        int
}

// Init initializes the model and loads projects from the database
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size first (applies to both screens)
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Calculate available space for list (subtract margins, status, help)
		listWidth := msg.Width - 4
		listHeight := msg.Height - 8

		if listHeight < 10 {
			listHeight = 10
		}

		m.list.SetSize(listWidth, listHeight)
	}

	// Handle setup screen
	if m.screen == screenSetupPath || m.screen == screenSetupGitHub || m.screen == screenOAuthWaiting {
		return m.updateSetup(msg)
	}

	// Handle cloud select screen
	if m.screen == screenCloudSelect {
		return m.updateCloudSelect(msg)
	}

	// Handle list screen
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If in clone input mode, only handle enter and esc
		if m.confirmClone {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				repoURL := m.cloneInput.Value()
				if repoURL == "" {
					m.errorMessage = "Please enter a valid GitHub repository URL"
					return m, nil
				}
				// Clear confirmation state
				m.confirmClone = false
				m.statusMessage = "Cloning repository..."
				m.errorMessage = ""
				// Execute clone
				return m, cloneProjectCmd(repoURL, m.rootScanPath)
			case "esc":
				m.confirmClone = false
				m.statusMessage = "Clone cancelled"
				m.errorMessage = ""
				return m, nil
			default:
				// Pass other keys to the text input
				var cmd tea.Cmd
				m.cloneInput, cmd = m.cloneInput.Update(msg)
				return m, cmd
			}
		}

		// If in archive confirmation mode, only handle enter and esc
		if m.confirmArchive {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				if m.archiveConfirmInput.Value() == "DELETE" {
					// Confirmed - proceed with archive
					originalItem := *m.archiveProject
					originalIdx := m.archiveIdx

					// OPTIMISTIC: Update the UI
					m.archiveProject.project.Status = "archived"
					m.archiveProject.isLoading = false
					m.list.SetItem(originalIdx, *m.archiveProject)

					// Clear confirmation state
					m.confirmArchive = false
					m.archiveProject = nil
					m.errorMessage = ""

					// Execute archive
					return m, archiveProjectCmd(originalItem.project.ID, originalItem, originalIdx)
				} else {
					// Wrong text typed
					m.errorMessage = "You must type 'DELETE' exactly to confirm"
					return m, nil
				}
			case "esc":
				m.confirmArchive = false
				m.archiveProject = nil
				m.statusMessage = "Archive cancelled"
				m.errorMessage = ""
				return m, nil
			default:
				// Pass other keys to the text input
				var cmd tea.Cmd
				m.archiveConfirmInput, cmd = m.archiveConfirmInput.Update(msg)
				return m, cmd
			}
		}

		// If list is filtering, let it handle all keys
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "d":
			// Archive (delete) the selected project - Show confirmation
			if m.confirmArchive {
				return m, nil // Already in confirmation mode
			}

			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			item, ok := selectedItem.(projectItem)
			if !ok {
				return m, nil
			}

			// Enter confirmation mode
			m.confirmArchive = true
			itemCopy := item
			m.archiveProject = &itemCopy
			m.archiveIdx = m.list.Index()
			m.errorMessage = ""
			m.statusMessage = ""

			// Create confirmation input
			confirmInput := textinput.New()
			confirmInput.Placeholder = "Type DELETE to confirm"
			confirmInput.Focus()
			confirmInput.CharLimit = 10
			confirmInput.Width = 30
			m.archiveConfirmInput = confirmInput

			return m, textinput.Blink

		case "r":
			// Restore the selected project - OPTIMISTIC UPDATE
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			item, ok := selectedItem.(projectItem)
			if !ok {
				return m, nil
			}

			// Only restore if it's archived
			if item.project.Status != "archived" {
				return m, nil
			}

			// Store original state for potential rollback
			originalItem := item
			originalIdx := m.list.Index()

			m.errorMessage = "" // Clear any previous errors
			m.statusMessage = "Restoring project..."

			// Return command to restore in background
			return m, restoreProjectCmd(item.project.ID, originalItem, originalIdx)

		case "enter":
			// Open project in VS Code
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			item, ok := selectedItem.(projectItem)
			if !ok {
				return m, nil
			}

			// Update LastOpened timestamp
			go db.UpdateLastOpened(item.project.ID)

			m.errorMessage = "" // Clear any previous errors

			// Return command to open VS Code
			return m, openProjectCmd(item.project.ID, item.project.Path)

		case "s":
			// Scan for new projects
			if m.isScanning {
				return m, nil // Already scanning
			}
			if m.rootScanPath == "" {
				m.errorMessage = "No scan path configured. Please restart."
				return m, nil
			}
			m.isScanning = true
			m.statusMessage = "Scanning for projects..."
			m.errorMessage = ""
			return m, scanProjectsWithPathCmd(m.rootScanPath)

		case "g":
			// Clone a GitHub repository
			if m.confirmClone {
				return m, nil // Already in clone mode
			}
			if m.rootScanPath == "" {
				m.errorMessage = "No scan path configured. Please restart."
				return m, nil
			}
			// Enter clone mode
			m.confirmClone = true
			m.errorMessage = ""
			m.statusMessage = ""

			// Create clone input
			cloneInput := textinput.New()
			cloneInput.Placeholder = "https://github.com/owner/repo"
			cloneInput.Focus()
			cloneInput.CharLimit = 256
			cloneInput.Width = 60
			m.cloneInput = cloneInput

			return m, textinput.Blink

		case "o":
			// Open GitHub repository URL in default browser
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			item, ok := selectedItem.(projectItem)
			if !ok {
				return m, nil
			}

			if item.project.RepoURL == "" {
				m.errorMessage = "No repository URL found for this project"
				return m, nil
			}

			m.errorMessage = "" // Clear any previous errors
			m.statusMessage = "Opening repository in browser..."

			// Open URL in default browser
			return m, openBrowserCmd(item.project.RepoURL)

		case "x":
			// Run/execute the selected project
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			item, ok := selectedItem.(projectItem)
			if !ok {
				return m, nil
			}

			m.errorMessage = "" // Clear any previous errors
			m.statusMessage = "Opening new terminal window to run project in development mode..."

			// Run the project
			return m, runProjectCmd(item.project.Path)

		case "c":
			// Clear all projects - ask for confirmation
			if !m.confirmClearAll {
				m.confirmClearAll = true
				m.errorMessage = ""
				m.statusMessage = ""
				return m, nil
			}
			// Confirmed - clear all
			m.confirmClearAll = false
			m.statusMessage = "Clearing all projects..."
			return m, clearAllProjectsCmd()

		case "u":
			// Check if GitHub token is configured
			if token, err := db.GetConfig("github_token"); err != nil || token == "" {
				m.errorMessage = "GitHub authentication required. Press 't' to authenticate with OAuth."
				return m, nil
			}
			// Sync to cloud (upload projects to GitHub Gist)
			m.errorMessage = ""
			m.statusMessage = "Syncing projects to cloud..."
			return m, syncToCloudCmd()

		case "l":
			// Check if GitHub token is configured
			if token, err := db.GetConfig("github_token"); err != nil || token == "" {
				m.errorMessage = "GitHub authentication required. Press 't' to authenticate with OAuth."
				return m, nil
			}
			// List projects from cloud
			m.errorMessage = ""
			m.statusMessage = "Loading projects from cloud..."
			return m, listCloudProjectsCmd()

		case "t":
			// Configure GitHub OAuth
			m.screen = screenSetupGitHub
			m.errorMessage = ""
			m.statusMessage = ""
			return m, nil

		case "esc":
			// Cancel clear all confirmation
			if m.confirmClearAll {
				m.confirmClearAll = false
				m.statusMessage = "Cancelled"
				return m, nil
			}
		}

	case ArchiveMsg:
		// Handle archive completion
		if msg.err != nil {
			// ROLLBACK: Archive failed, revert the change
			m.list.SetItem(msg.originalIdx, msg.originalItem)
			m.errorMessage = fmt.Sprintf("Archive failed: %v", msg.err)
			return m, nil
		} else {
			// Success: Reload list from database to fix filtering and prevent duplicates
			m.errorMessage = ""
			m.statusMessage = "Project archived successfully"
			return m, reloadProjectsCmd()
		}

	case RestoreMsg:
		// Handle restore completion
		if msg.err != nil {
			// ROLLBACK: Restore failed, revert the change
			m.list.SetItem(msg.originalIdx, msg.originalItem)
			m.errorMessage = fmt.Sprintf("Restore failed: %v", msg.err)
			return m, nil
		} else {
			// SUCCESS: Reload list from database to fix filtering and prevent duplicates
			m.errorMessage = ""
			m.statusMessage = "Project restored successfully"
			return m, reloadProjectsCmd()
		}

	case CloneMsg:
		// Handle clone completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Clone failed: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.errorMessage = ""
			m.statusMessage = fmt.Sprintf("Successfully cloned %s", msg.projectName)
			// Reload the list to show the new project
			return m, reloadProjectsCmd()
		}
		return m, nil

	case OpenProjectMsg:
		// Handle VS Code open completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to open VS Code: %v", msg.err)
		} else {
			m.errorMessage = "" // Clear error on success
		}
		return m, nil

	case OpenBrowserMsg:
		// Handle browser open completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to open browser: %v", msg.err)
		} else {
			m.errorMessage = "" // Clear error on success
			m.statusMessage = "Repository opened in browser"
		}
		return m, nil

	case RunProjectMsg:
		// Handle project run completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to open terminal: %v", msg.err)
		} else {
			m.errorMessage = "" // Clear error on success
			m.statusMessage = "Development terminal opened - project is running in dev mode"
		}
		return m, nil

	case ScanCompleteMsg:
		// Handle scan completion
		m.isScanning = false
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Scan failed: %v", msg.err)
			m.statusMessage = ""
		} else {
			if msg.projectsRemoved > 0 {
				m.statusMessage = fmt.Sprintf("Scan complete: Found %d, added %d new, removed %d", msg.projectsFound, msg.projectsAdded, msg.projectsRemoved)
			} else {
				m.statusMessage = fmt.Sprintf("Scan complete: Found %d projects, added %d new", msg.projectsFound, msg.projectsAdded)
			}
			m.errorMessage = ""
			// Switch to list view if we're on setup screen
			if m.screen == screenSetupPath || m.screen == screenSetupGitHub {
				m.screen = screenList
			}
			// Reload the list
			return m, reloadProjectsCmd()
		}
		return m, nil

	case ClearAllMsg:
		// Handle clear all completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to clear projects: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.statusMessage = fmt.Sprintf("Cleared %d projects from database", msg.count)
			m.errorMessage = ""
			// Clear the list
			m.list.SetItems([]list.Item{})
			// Switch to setup screen
			m.screen = screenSetupPath

			// Create and focus new path input
			ti := textinput.New()
			ti.Placeholder = "Enter path (e.g., D:\\\\Projects)"
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 60

			// Reset path input with stored or home directory
			if m.rootScanPath != "" {
				ti.SetValue(m.rootScanPath)
			} else if homeDir, err := os.UserHomeDir(); err == nil {
				ti.SetValue(homeDir)
			}

			m.pathInput = ti
			return m, textinput.Blink
		}
		return m, nil

	case reloadMsg:
		// Reload the list with new items
		m.list.SetItems(msg.items)
		return m, nil

	case SyncToCloudMsg:
		// Handle sync to cloud completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Sync to cloud failed: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.errorMessage = ""
			m.statusMessage = fmt.Sprintf("Projects synced to cloud (Gist ID: %s)", msg.gistID)
			// Save the gist ID to config
			go db.SetConfig("gist_id", msg.gistID)
		}
		return m, nil

	case LoadFromCloudMsg:
		// Handle load from cloud completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Load from cloud failed: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.errorMessage = ""
			m.statusMessage = fmt.Sprintf("Loaded %d projects from cloud", msg.projectsLoaded)
			// Reload the list to show loaded projects
			return m, reloadProjectsCmd()
		}
		return m, nil

	case ListCloudProjectsMsg:
		// Handle list cloud projects completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to list cloud projects: %v", msg.err)
			m.statusMessage = ""
			return m, nil
		}
		m.cloudProjects = msg.projects
		m.selectedCloudIndices = []int{}
		m.cloudCursorIndex = 0 // Initialize cursor at first item
		m.screen = screenCloudSelect
		m.statusMessage = ""
		m.errorMessage = ""
		return m, nil

	case LoadSelectedProjectsMsg:
		// Handle load selected projects completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to load selected projects: %v", msg.err)
			m.statusMessage = ""
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Loaded %d projects from cloud (marked as archived)", msg.projectsLoaded)
		m.errorMessage = ""
		// Reload the list to show the new archived projects
		return m, reloadProjectsCmd()

	case ErrorMsg:
		m.errorMessage = msg.err.Error()
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateSetup handles updates for the setup screen
func (m model) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.screen == screenSetupPath {
				// Handle path input
				if m.pathInput.Value() == "" {
					m.errorMessage = "Please enter a valid path"
					return m, nil
				}
				m.isScanning = true
				m.statusMessage = "Scanning for projects..."
				m.errorMessage = ""
				m.rootScanPath = m.pathInput.Value()
				// Save root path to config
				_ = db.SetConfig("root_scan_path", m.pathInput.Value())
				return m, scanProjectsWithPathCmd(m.pathInput.Value())
			} else if m.screen == screenSetupGitHub {
				// User pressed enter to start OAuth flow
				m.statusMessage = "Initiating GitHub authentication..."
				m.errorMessage = ""
				return m, initiateOAuthCmd()
			} else if m.screen == screenSetupToken {
				// Handle token input
				token := m.tokenInput.Value()
				if token == "" {
					m.errorMessage = "Please enter a valid GitHub token"
					return m, nil
				}

				// Validate token before saving
				validationClient := engine.NewGistClient(token)
				if err := validationClient.ValidateToken(); err != nil {
					m.errorMessage = "Invalid GitHub token. Please check your token and try again."
					return m, nil
				}

				// Save token to config
				_ = db.SetConfig("github_token", token)
				m.statusMessage = "GitHub token configured successfully"
				m.errorMessage = ""
				m.screen = screenList
				return m, reloadProjectsCmd()
			}
		default:
			// For any other key, pass it to the appropriate text input
			var cmd tea.Cmd
			if m.screen == screenSetupPath {
				m.pathInput, cmd = m.pathInput.Update(msg)
			} else if m.screen == screenSetupToken {
				if msg.String() == "esc" {
					// Go back to GitHub setup screen
					m.screen = screenSetupGitHub
					m.errorMessage = ""
					m.statusMessage = ""
					return m, nil
				}
				m.tokenInput, cmd = m.tokenInput.Update(msg)
			} else if m.screen == screenSetupGitHub {
				// On GitHub setup screen, handle skip or PAT option
				if msg.String() == "s" {
					// Skip OAuth setup
					m.screen = screenList
					m.statusMessage = "Skipped GitHub authentication. You can configure it later with 't'."
					return m, reloadProjectsCmd()
				} else if msg.String() == "p" {
					// Switch to manual token entry
					m.screen = screenSetupToken
					m.errorMessage = ""
					m.statusMessage = ""
					// Initialize token input
					ti := textinput.New()
					ti.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxx"
					ti.Focus()
					ti.CharLimit = 256
					ti.Width = 60
					m.tokenInput = ti
					return m, textinput.Blink
				}
			}
			return m, cmd
		}

	case ScanCompleteMsg:
		// Handle scan completion
		m.isScanning = false
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Scan failed: %v", msg.err)
			m.statusMessage = ""
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Found %d projects, added %d to database", msg.projectsFound, msg.projectsAdded)
		// Switch to GitHub setup screen
		m.screen = screenSetupGitHub
		m.errorMessage = ""
		return m, nil

	case OAuthDeviceCodeMsg:
		// Handle device code response
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("OAuth initiation failed: %v", msg.err)
			m.statusMessage = "Falling back to manual token entry..."
			// Automatically switch to manual token entry after a short delay
			m.screen = screenSetupToken
			// Initialize token input
			ti := textinput.New()
			ti.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxx"
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 60
			m.tokenInput = ti
			return m, textinput.Blink
		}
		m.oauthDeviceCode = msg.deviceCode
		m.oauthUserCode = msg.userCode
		m.oauthVerificationURI = msg.verificationURI
		m.oauthInterval = msg.interval
		m.screen = screenOAuthWaiting
		m.statusMessage = "Waiting for authentication..."
		m.errorMessage = ""
		// Start polling for access token
		return m, pollForAccessTokenCmd(msg.deviceCode, msg.interval)

	case OAuthCompleteMsg:
		// Handle OAuth completion
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("OAuth failed: %v", msg.err)
			m.statusMessage = "Falling back to manual token entry..."
			// Automatically switch to manual token entry after a short delay
			m.screen = screenSetupToken
			// Initialize token input
			ti := textinput.New()
			ti.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxx"
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 60
			m.tokenInput = ti
			return m, textinput.Blink
		}
		// Save token to config
		_ = db.SetConfig("github_token", msg.accessToken)
		m.statusMessage = "GitHub authentication successful!"
		m.errorMessage = ""
		m.screen = screenList
		return m, reloadProjectsCmd()

	case reloadMsg:
		// Load projects into list and switch to list screen
		m.list.SetItems(msg.items)
		m.screen = screenList
		return m, nil
	}

	return m, nil
}

// updateCloudSelect handles updates for the cloud project selection screen
func (m model) updateCloudSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle filter input when filtering mode is active
		if m.cloudFiltering {
			switch msg.String() {
			case "esc":
				// Exit filter mode
				m.cloudFiltering = false
				m.cloudFilterInput.Blur()
				m.cloudFilterInput.SetValue("")
				m.cloudCursorIndex = 0
				m.errorMessage = ""
				return m, nil
			case "enter":
				// Exit filter mode and keep the filter
				m.cloudFiltering = false
				m.cloudFilterInput.Blur()
				m.cloudCursorIndex = 0
				m.errorMessage = ""
				return m, nil
			default:
				// Update filter input
				var cmd tea.Cmd
				m.cloudFilterInput, cmd = m.cloudFilterInput.Update(msg)
				// Reset cursor when filter changes
				m.cloudCursorIndex = 0
				m.errorMessage = ""
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.screen = screenList
			m.cloudProjects = nil
			m.selectedCloudIndices = nil
			m.cloudCursorIndex = 0
			m.cloudFiltering = false
			m.cloudFilterInput.SetValue("")
			return m, nil

		case "enter":
			if len(m.selectedCloudIndices) == 0 {
				m.errorMessage = "Please select at least one project"
				return m, nil
			}
			// Load selected projects
			return m, loadSelectedProjectsCmd(m.selectedCloudIndices, m.cloudProjects)

		case "esc":
			m.screen = screenList
			m.cloudProjects = nil
			m.selectedCloudIndices = nil
			m.cloudCursorIndex = 0
			m.cloudFiltering = false
			m.cloudFilterInput.SetValue("")
			return m, nil

		case "/":
			// Enter filter mode
			m.cloudFiltering = true
			m.cloudFilterInput.Focus()
			m.errorMessage = ""
			return m, textinput.Blink

		case "up", "k":
			// Move cursor up in filtered list
			filteredIndices := m.getFilteredIndices()
			if len(filteredIndices) == 0 {
				return m, nil
			}

			// Find current position in filtered list
			currentPos := -1
			for i, idx := range filteredIndices {
				if idx == m.cloudCursorIndex {
					currentPos = i
					break
				}
			}

			// Move to previous item in filtered list
			if currentPos > 0 {
				m.cloudCursorIndex = filteredIndices[currentPos-1]
			} else if currentPos == -1 && len(filteredIndices) > 0 {
				// Cursor not in filtered list, move to last filtered item
				m.cloudCursorIndex = filteredIndices[len(filteredIndices)-1]
			}
			m.errorMessage = ""
			return m, nil

		case "down", "j":
			// Move cursor down in filtered list
			filteredIndices := m.getFilteredIndices()
			if len(filteredIndices) == 0 {
				return m, nil
			}

			// Find current position in filtered list
			currentPos := -1
			for i, idx := range filteredIndices {
				if idx == m.cloudCursorIndex {
					currentPos = i
					break
				}
			}

			// Move to next item in filtered list
			if currentPos >= 0 && currentPos < len(filteredIndices)-1 {
				m.cloudCursorIndex = filteredIndices[currentPos+1]
			} else if currentPos == -1 && len(filteredIndices) > 0 {
				// Cursor not in filtered list, move to first filtered item
				m.cloudCursorIndex = filteredIndices[0]
			}
			m.errorMessage = ""
			return m, nil

		case "pgup":
			// Jump up by 10 items in filtered list
			filteredIndices := m.getFilteredIndices()
			if len(filteredIndices) == 0 {
				return m, nil
			}

			currentPos := -1
			for i, idx := range filteredIndices {
				if idx == m.cloudCursorIndex {
					currentPos = i
					break
				}
			}

			if currentPos >= 0 {
				newPos := max(0, currentPos-10)
				m.cloudCursorIndex = filteredIndices[newPos]
			} else if len(filteredIndices) > 0 {
				m.cloudCursorIndex = filteredIndices[len(filteredIndices)-1]
			}
			m.errorMessage = ""
			return m, nil

		case "pgdown":
			// Jump down by 10 items in filtered list
			filteredIndices := m.getFilteredIndices()
			if len(filteredIndices) == 0 {
				return m, nil
			}

			currentPos := -1
			for i, idx := range filteredIndices {
				if idx == m.cloudCursorIndex {
					currentPos = i
					break
				}
			}

			if currentPos >= 0 {
				newPos := min(len(filteredIndices)-1, currentPos+10)
				m.cloudCursorIndex = filteredIndices[newPos]
			} else if len(filteredIndices) > 0 {
				m.cloudCursorIndex = filteredIndices[0]
			}
			m.errorMessage = ""
			return m, nil

		case "home", "g":
			// Jump to first item in filtered list
			filteredIndices := m.getFilteredIndices()
			if len(filteredIndices) > 0 {
				m.cloudCursorIndex = filteredIndices[0]
			}
			m.errorMessage = ""
			return m, nil

		case "end", "G":
			// Jump to last item in filtered list
			filteredIndices := m.getFilteredIndices()
			if len(filteredIndices) > 0 {
				m.cloudCursorIndex = filteredIndices[len(filteredIndices)-1]
			}
			m.errorMessage = ""
			return m, nil

		case " ", "tab":
			// Toggle selection at current cursor position
			idx := m.cloudCursorIndex
			found := false
			for i, selectedIdx := range m.selectedCloudIndices {
				if selectedIdx == idx {
					// Remove from selection
					m.selectedCloudIndices = append(m.selectedCloudIndices[:i], m.selectedCloudIndices[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				// Add to selection
				m.selectedCloudIndices = append(m.selectedCloudIndices, idx)
			}
			m.errorMessage = ""
			return m, nil

		case "a":
			// Select all filtered projects
			filteredIndices := m.getFilteredIndices()
			m.selectedCloudIndices = filteredIndices
			m.errorMessage = ""
			if len(filteredIndices) == len(m.cloudProjects) {
				m.statusMessage = fmt.Sprintf("Selected all %d projects", len(filteredIndices))
			} else {
				m.statusMessage = fmt.Sprintf("Selected all %d filtered projects", len(filteredIndices))
			}
			return m, nil

		case "n":
			// Clear all selections
			m.selectedCloudIndices = nil
			m.errorMessage = ""
			m.statusMessage = "Cleared all selections"
			return m, nil

		case "i":
			// Invert selection
			newSelection := []int{}
			for i := range m.cloudProjects {
				isSelected := false
				for _, selectedIdx := range m.selectedCloudIndices {
					if selectedIdx == i {
						isSelected = true
						break
					}
				}
				if !isSelected {
					newSelection = append(newSelection, i)
				}
			}
			m.selectedCloudIndices = newSelection
			m.errorMessage = ""
			m.statusMessage = fmt.Sprintf("Inverted selection (%d selected)", len(newSelection))
			return m, nil

		default:
			// Handle number keys for quick selection (1-9) - legacy support
			if len(msg.String()) == 1 {
				num := int(msg.String()[0] - '0')
				if num >= 1 && num <= min(9, len(m.cloudProjects)) {
					idx := num - 1
					// Toggle selection
					found := false
					for i, selectedIdx := range m.selectedCloudIndices {
						if selectedIdx == idx {
							// Remove from selection
							m.selectedCloudIndices = append(m.selectedCloudIndices[:i], m.selectedCloudIndices[i+1:]...)
							found = true
							break
						}
					}
					if !found {
						// Add to selection
						m.selectedCloudIndices = append(m.selectedCloudIndices, idx)
					}
					m.errorMessage = ""
					return m, nil
				}
			}
		}

	case LoadSelectedProjectsMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("Failed to load selected projects: %v", msg.err)
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Loaded %d projects from cloud (marked as archived)", msg.projectsLoaded)
		m.errorMessage = ""
		m.screen = screenList
		m.cloudProjects = nil
		m.selectedCloudIndices = nil
		m.cloudCursorIndex = 0
		// Reload the list to show the new archived projects
		return m, reloadProjectsCmd()
	}

	return m, nil
}

// View renders the UI
func (m model) View() string {
	if m.screen == screenSetupPath || m.screen == screenSetupGitHub || m.screen == screenSetupToken || m.screen == screenOAuthWaiting {
		return m.viewSetup()
	}
	if m.screen == screenCloudSelect {
		return m.viewCloudSelect()
	}
	return m.viewList()
}

// viewSetup renders the setup screen
func (m model) viewSetup() string {
	var s string

	if m.screen == screenSetupPath {
		// Title
		title := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")).
			Bold(true).
			Render("\n╔═══════════════════════════════════════════════════════════╗\n" +
				"║              Welcome to DevBase v1.0.0                  ║\n" +
				"╚═══════════════════════════════════════════════════════════╝\n")
		s += title

		// Prompt
		prompt := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Render("\nEnter the root folder path for your projects:\n")
		s += prompt

		// Hint
		hint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Render("(e.g., D:\\\\Projects or C:\\\\Users\\\\YourName\\\\workspace)\n\n")
		s += hint

		// Input field
		s += m.pathInput.View() + "\n"

		// Help text
		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("\nPress Enter to start scan | Ctrl+C to quit")
		s += helpText

	} else if m.screen == screenSetupGitHub {
		// Title box with consistent styling
		titleBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FFFF")).
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			Render("Configure GitHub Integration")

		s += "\n" + titleBox + "\n\n"

		// Authentication options
		oauthBox := lipgloss.NewStyle().
			Width(58).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#00FF00")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("Option 1: OAuth Device Flow (Recommended)") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("• Secure browser-based authentication") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("• No manual token creation needed") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("• Automatic token management") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press ENTER to start OAuth flow"),
			)

		s += oauthBox + "\n\n"

		patBox := lipgloss.NewStyle().
			Width(58).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FFFF00")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true).Render("Option 2: Personal Access Token") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("• Manual token creation required") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("• Create token at github.com/settings/tokens") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("• Requires 'gist' scope only") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press P for manual token entry"),
			)

		s += patBox + "\n\n"

		// Help text
		skipBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("Press S to skip setup  •  Ctrl+C to quit")

		s += skipBox

	} else if m.screen == screenSetupToken {
		// Title box
		titleBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFFF00")).
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#FFFF00")).
			Render("Enter GitHub Personal Access Token")

		s += "\n" + titleBox + "\n\n"

		// Instructions
		instructions := lipgloss.NewStyle().
			Width(60).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("Create a Personal Access Token:") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render("1. Visit: https://github.com/settings/tokens") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render("2. Click 'Generate new token (classic)'") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render("3. Select only 'gist' scope") + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render("4. Copy the token and paste below") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Token will be stored securely in your local database."),
			)

		s += instructions + "\n\n"

		// Input field
		s += m.tokenInput.View() + "\n\n"

		// Help text
		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("Press Enter to save token  •  Press Esc to go back  •  Ctrl+C to quit")

		s += helpText

	} else if m.screen == screenOAuthWaiting {
		// Title box
		titleBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FFFF")).
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			Render("GitHub Authentication in Progress")

		s += "\n" + titleBox + "\n\n"

		// Instructions header
		instructionsHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Render("Please complete the following steps:")

		s += instructionsHeader + "\n\n"

		// Step 1 - Visit URL (in a box for emphasis)
		step1Box := lipgloss.NewStyle().
			Width(60).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#00FF00")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("STEP 1: Visit this URL") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(m.oauthVerificationURI),
			)

		s += step1Box + "\n\n"

		// Step 2 - Enter code (highlighted box)
		step2Box := lipgloss.NewStyle().
			Width(60).
			Padding(1, 2).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FFFF00")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true).Render("STEP 2: Enter this code") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true).Render(m.oauthUserCode),
			)

		s += step2Box + "\n\n"

		// Step 3 - Authorize
		step3Box := lipgloss.NewStyle().
			Width(60).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#00FF00")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("STEP 3: Authorize DevBase") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Grant DevBase access to your Gists"),
			)

		s += step3Box + "\n\n"

		// Waiting indicator with animation suggestion
		waitingMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")).
			Bold(true).
			Render("⟳ Waiting for authorization...")

		waitingSubtext := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Render("This window will automatically continue once you authorize")

		s += waitingMsg + "\n" + waitingSubtext
	}

	// Display error message if present
	if m.errorMessage != "" {
		errorView := errorStyle.Render("\n⚠ " + m.errorMessage)
		s += errorView
	}

	// Display status message if present
	if m.statusMessage != "" {
		statusView := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00")).
			Render("\n✓ " + m.statusMessage)
		s += statusView
	}

	// Add scanning indicator
	if m.isScanning {
		scanIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true).
			Render("\n\n⟳ Scanning directories...")
		s += scanIndicator
	}

	return docStyle.Render(s)
}

// viewCloudSelect renders the cloud project selection screen
func (m model) viewCloudSelect() string {
	// Title box
	titleBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FFFF")).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Render("Select Projects from Cloud")

	s := "\n" + titleBox + "\n\n"

	// Instructions box
	instructionsBox := lipgloss.NewStyle().
		Width(68).
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Render(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("Select projects to load from cloud") + "\n" +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Selected projects will be marked as archived for safety"),
		)
	s += instructionsBox + "\n\n"

	// Show filter input if filtering is active
	if m.cloudFiltering {
		filterBox := lipgloss.NewStyle().
			Width(68).
			Padding(0, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#00FFFF")).
			Render(m.cloudFilterInput.View())
		s += filterBox + "\n\n"
	}

	// Apply filter to projects and track original indices
	type filteredProject struct {
		project     models.Project
		originalIdx int
	}

	filteredProjects := []filteredProject{}
	filterText := strings.ToLower(strings.TrimSpace(m.cloudFilterInput.Value()))

	for i, project := range m.cloudProjects {
		if filterText == "" {
			filteredProjects = append(filteredProjects, filteredProject{project: project, originalIdx: i})
		} else if strings.Contains(strings.ToLower(project.Name), filterText) ||
			strings.Contains(strings.ToLower(project.Path), filterText) ||
			strings.Contains(strings.ToLower(project.RepoURL), filterText) {
			filteredProjects = append(filteredProjects, filteredProject{project: project, originalIdx: i})
		}
	}

	// Calculate max name length for proper alignment
	maxNameLen := 0
	maxNumberLen := len(fmt.Sprintf("%d", len(filteredProjects)))
	for _, fp := range filteredProjects {
		if len(fp.project.Name) > maxNameLen {
			maxNameLen = len(fp.project.Name)
		}
	}

	// Project list container with count
	projectCountInfo := ""
	if filterText != "" {
		projectCountInfo = fmt.Sprintf(" (%d of %d)", len(filteredProjects), len(m.cloudProjects))
	}
	projectListHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Bold(true).
		Render("Available Projects:" + projectCountInfo)
	s += projectListHeader + "\n\n"

	// If no projects match filter
	if len(filteredProjects) == 0 {
		noResultsMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("  No projects match the filter")
		s += noResultsMsg + "\n"
	}

	// List cloud projects with aligned formatting and cursor
	for i, fp := range filteredProjects {
		originalIdx := fp.originalIdx
		project := fp.project

		isSelected := false
		for _, idx := range m.selectedCloudIndices {
			if idx == originalIdx {
				isSelected = true
				break
			}
		}

		isCursor := (originalIdx == m.cloudCursorIndex)

		// Build the line with proper alignment
		checkbox := "[ ]"
		if isSelected {
			checkbox = "[✓]"
		}

		// Cursor indicator
		cursor := " "
		if isCursor {
			cursor = "►"
		}

		number := fmt.Sprintf("%*d.", maxNumberLen, i+1)
		projectName := fmt.Sprintf("%-*s", maxNameLen, project.Name)

		// Additional info if available
		var additionalInfo string
		if project.RepoURL != "" {
			iconColor := "#666666"
			if isCursor {
				iconColor = "#00FFFF"
			}
			additionalInfo = lipgloss.NewStyle().
				Foreground(lipgloss.Color(iconColor)).
				Render(" 🔗")
		}

		// Style based on cursor position and selection
		lineStyle := lipgloss.NewStyle()

		if isCursor && isSelected {
			// Cursor on selected item
			lineStyle = lineStyle.
				Background(lipgloss.Color("#00AA00")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)
		} else if isCursor {
			// Cursor on unselected item
			lineStyle = lineStyle.
				Background(lipgloss.Color("#444444")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
		} else if isSelected {
			// Selected but not cursor
			lineStyle = lineStyle.
				Foreground(lipgloss.Color("#00FF00")).
				Bold(true)
		} else {
			// Normal item
			lineStyle = lineStyle.
				Foreground(lipgloss.Color("#FFFFFF"))
		}

		line := fmt.Sprintf("%s %s %s %s%s", cursor, checkbox, number, projectName, additionalInfo)
		s += lineStyle.Render(line) + "\n"
	}

	// Selection summary
	if len(m.selectedCloudIndices) > 0 {
		summaryBox := lipgloss.NewStyle().
			MarginTop(1).
			Padding(0, 2).
			Foreground(lipgloss.Color("#00FF00")).
			Render(fmt.Sprintf("✓ %d project(s) selected", len(m.selectedCloudIndices)))
		s += "\n" + summaryBox + "\n"
	} else {
		summaryBox := lipgloss.NewStyle().
			MarginTop(1).
			Padding(0, 2).
			Foreground(lipgloss.Color("#888888")).
			Render("No projects selected")
		s += "\n" + summaryBox + "\n"
	}

	// Compact help text - single line format
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("\n↑↓/jk=navigate  space=toggle  /=filter  a=all  n=none  enter=load  esc=cancel")
	s += helpText

	// Display error message if present
	if m.errorMessage != "" {
		errorView := errorStyle.Render(fmt.Sprintf("\n⚠ %s", m.errorMessage))
		s += errorView
	}

	// Display status message if present
	if m.statusMessage != "" {
		statusView := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00")).
			Render("\n✓ " + m.statusMessage)
		s += statusView
	}

	return docStyle.Render(s)
}

// viewList renders the project list screen
func (m model) viewList() string {
	// If not ready, show loading state
	if !m.ready {
		return "Loading..."
	}

	view := m.list.View()

	// Add token status indicator
	var tokenStatus string
	if token, err := db.GetConfig("github_token"); err != nil || token == "" {
		tokenStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Render("\n☁ Cloud sync disabled - GitHub OAuth not configured (press 't' to authenticate)")
	} else {
		tokenStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00")).
			Render("\n☁ Cloud sync enabled (authenticated)")
	}
	view += tokenStatus

	// Display error message if present
	if m.errorMessage != "" {
		errorView := errorStyle.Render(fmt.Sprintf("\n⚠ %s", m.errorMessage))
		view += errorView
	}

	// Add scanning indicator
	scanIndicator := ""
	if m.isScanning {
		scanIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true).
			Render("\n\n⟳ Scanning directories...")
	}

	// Add status message
	statusView := ""
	if m.statusMessage != "" {
		statusView = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00")).
			Render("\n\n✓ " + m.statusMessage)
	}

	// Add clone input dialog if in clone mode
	clonePrompt := ""
	if m.confirmClone {
		clonePrompt = "\n\n" +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FFFF")).
				Bold(true).
				Render("🔗 CLONE GITHUB REPOSITORY") + "\n\n" +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Render("Enter GitHub repository URL:") + "\n" +
			m.cloneInput.View() + "\n\n" +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Render("Press Enter to clone | ESC to cancel")
	}

	// Add archive confirmation dialog if in archive mode
	archivePrompt := ""
	if m.confirmArchive && m.archiveProject != nil {
		hasRepoURL := m.archiveProject.project.RepoURL != ""

		// Warning title box
		warningTitle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF0000")).
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#FF0000")).
			Render("⚠ WARNING: ARCHIVE PROJECT")

		archivePrompt = "\n\n" + warningTitle + "\n\n"

		// Project information box
		projectInfoBox := lipgloss.NewStyle().
			Width(70).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render("Project Details:") + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render("Name: ") +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(m.archiveProject.project.Name) + "\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render("Path: ") +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(m.archiveProject.project.Path),
			)

		archivePrompt += projectInfoBox + "\n\n"

		// Restore capability box
		if hasRepoURL {
			restoreBox := lipgloss.NewStyle().
				Width(70).
				Padding(1, 2).
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#00FF00")).
				Render(
					lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("✓ Restore Available") + "\n\n" +
						lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("This project can be restored from:") + "\n" +
						lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render(m.archiveProject.project.RepoURL),
				)
			archivePrompt += restoreBox + "\n\n"
		} else {
			warningBox := lipgloss.NewStyle().
				Width(70).
				Padding(1, 2).
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#FFAA00")).
				Render(
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Bold(true).Render("⚠ PERMANENT DELETION WARNING") + "\n\n" +
						lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("No git repository URL found!") + "\n" +
						lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("This project CANNOT be restored after archiving.") + "\n" +
						lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("All files will be permanently deleted."),
				)
			archivePrompt += warningBox + "\n\n"
		}

		// Confirmation input box
		confirmBox := lipgloss.NewStyle().
			Width(70).
			Padding(1, 2).
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#FF0000")).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("Type 'DELETE' to confirm:") + "\n\n" +
					m.archiveConfirmInput.View() + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press Enter to confirm  •  ESC to cancel"),
			)

		archivePrompt += confirmBox
	}

	// Add confirmation prompt if in clear all mode
	confirmPrompt := ""
	if m.confirmClearAll {
		confirmPrompt = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Bold(true).
			Render("\n\n⚠ WARNING: Clear ALL projects from database?\n") +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Render("Press C again to CONFIRM | ESC to Cancel")
	}

	// Add help text
	var helpText string
	if token, err := db.GetConfig("github_token"); err != nil || token == "" {
		// Token not configured
		helpText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("\n\nKeys: enter=open  o=browser  x=run  s=scan  g=clone  t=github-oauth  c=clear-all  d=archive  r=restore  /=filter  q=quit")
	} else {
		// Token configured
		helpText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("\n\nKeys: enter=open  o=browser  x=run  s=scan  g=clone  u=sync-up  l=select-cloud  t=github-oauth  c=clear-all  d=archive  r=restore  /=filter  q=quit")
	}

	// Build output without extra docStyle wrapping to avoid layout issues
	return view + scanIndicator + statusView + clonePrompt + archivePrompt + confirmPrompt + helpText
}

// NewModel creates a new model with projects loaded from the database
func NewModel() (model, error) {
	// Load projects from the database
	projects, err := db.GetProjects()
	if err != nil {
		return model{}, fmt.Errorf("failed to load projects: %w", err)
	}

	// Load root scan path from config
	rootPath, _ := db.GetConfig("root_scan_path")

	// Create the list with reasonable default dimensions
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "DevBase - Project Manager"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	// If database is empty, start with setup screen
	if len(projects) == 0 {
		// Create text input for path
		ti := textinput.New()
		ti.Placeholder = "Enter path (e.g., D:\\\\Projects)"
		ti.Focus()
		ti.CharLimit = 256
		ti.Width = 60

		// Get user home directory as default
		if homeDir, err := os.UserHomeDir(); err == nil {
			ti.SetValue(homeDir)
		}

		// Create cloud filter input
		cloudFilter := textinput.New()
		cloudFilter.Placeholder = "Type to filter projects..."
		cloudFilter.CharLimit = 100
		cloudFilter.Width = 50

		return model{
			screen:               screenSetupPath,
			pathInput:            ti,
			tokenInput:           textinput.New(),
			list:                 l,
			errorMessage:         "",
			statusMessage:        "",
			isScanning:           false,
			confirmClearAll:      false,
			confirmArchive:       false,
			confirmClone:         false,
			cloneInput:           textinput.New(),
			cloudProjects:        nil,
			selectedCloudIndices: nil,
			cloudCursorIndex:     0,
			cloudFilterInput:     cloudFilter,
			cloudFiltering:       false,
			rootScanPath:         rootPath,
			width:                80,
			height:               24,
			ready:                false,
		}, nil
	}

	// Convert projects to list items
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p, isLoading: false}
	}
	l.SetItems(items)

	// Create cloud filter input
	cloudFilter := textinput.New()
	cloudFilter.Placeholder = "Type to filter projects..."
	cloudFilter.CharLimit = 100
	cloudFilter.Width = 50

	return model{
		screen:               screenList,
		pathInput:            textinput.New(),
		tokenInput:           textinput.New(),
		list:                 l,
		errorMessage:         "",
		statusMessage:        "",
		isScanning:           false,
		confirmClearAll:      false,
		confirmArchive:       false,
		confirmClone:         false,
		cloneInput:           textinput.New(),
		cloudProjects:        nil,
		selectedCloudIndices: nil,
		cloudCursorIndex:     0,
		cloudFilterInput:     cloudFilter,
		cloudFiltering:       false,
		rootScanPath:         rootPath,
		width:                80,
		height:               24,
		ready:                false,
	}, nil
}

// archiveProjectCmd creates a command that archives a project in the background
func archiveProjectCmd(projectID uint, originalItem projectItem, originalIdx int) tea.Cmd {
	return func() tea.Msg {
		err := engine.ArchiveProject(projectID)
		return ArchiveMsg{
			projectID:    projectID,
			err:          err,
			originalItem: originalItem,
			originalIdx:  originalIdx,
		}
	}
}

// restoreProjectCmd creates a command that restores a project in the background
func restoreProjectCmd(projectID uint, originalItem projectItem, originalIdx int) tea.Cmd {
	return func() tea.Msg {
		err := engine.RestoreProject(projectID)
		return RestoreMsg{
			projectID:    projectID,
			err:          err,
			originalItem: originalItem,
			originalIdx:  originalIdx,
		}
	}
}

// openProjectCmd creates a command that opens a project in VS Code
func openProjectCmd(projectID uint, path string) tea.Cmd {
	return func() tea.Msg {
		// Open VS Code with the project path
		cmd := exec.Command("code", path)
		err := cmd.Start()
		return OpenProjectMsg{
			projectID: projectID,
			err:       err,
		}
	}
}

// openBrowserCmd creates a command that opens a URL in the default browser
func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		// Open URL in default browser using PowerShell's Start-Process
		cmd := exec.Command("powershell", "-Command", "Start-Process", url)
		err := cmd.Start()
		return OpenBrowserMsg{
			url: url,
			err: err,
		}
	}
}

// runProjectCmd creates a command that runs/executes a project in a new terminal window
func runProjectCmd(projectPath string) tea.Cmd {
	return func() tea.Msg {
		// Detect project type and get the run command
		cmd, err := detectAndCreateRunCommand(projectPath)
		if err != nil {
			return RunProjectMsg{
				projectPath: projectPath,
				err:         err,
			}
		}

		// Build the full command string
		args := strings.Join(cmd.Args, " ")
		fullCommand := fmt.Sprintf("cd /d %s && %s", projectPath, args)

		// Open new terminal window with the command
		// Use cmd /c start cmd /k to open a new cmd window that stays open
		terminalCmd := exec.Command("cmd", "/c", "start", "cmd", "/k", fullCommand)

		err = terminalCmd.Start()
		return RunProjectMsg{
			projectPath: projectPath,
			err:         err,
		}
	}
}

// getNpmDevCommand checks if package.json has a dev script and returns appropriate npm command
func getNpmDevCommand(projectPath string) string {
	packageJsonPath := filepath.Join(projectPath, "package.json")

	// Read package.json
	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return "npm start" // fallback
	}

	// Simple check for "dev" script - look for "dev" in scripts section
	contentStr := string(content)
	if strings.Contains(contentStr, `"dev"`) && strings.Contains(contentStr, `"scripts"`) {
		return "npm run dev"
	}

	return "npm start" // fallback to start
}

// getPythonDevCommand checks for Python framework specific development commands
func getPythonDevCommand(projectPath string) string {
	// Check for Django manage.py
	if _, err := os.Stat(filepath.Join(projectPath, "manage.py")); err == nil {
		return "python manage.py runserver"
	}

	// Check for Flask app.py with debug mode
	if _, err := os.Stat(filepath.Join(projectPath, "app.py")); err == nil {
		return "python -c \"from app import app; app.run(debug=True)\""
	}

	// Check for main.py
	if _, err := os.Stat(filepath.Join(projectPath, "main.py")); err == nil {
		return "python main.py"
	}

	// Fallback
	return "python -m main"
}

// detectAndCreateRunCommand detects project type and creates appropriate run command
func detectAndCreateRunCommand(projectPath string) (*exec.Cmd, error) {
	// Check for Go project
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		// Go project - install dependencies and run
		mainFiles, err := filepath.Glob(filepath.Join(projectPath, "cmd", "*", "main.go"))
		if err == nil && len(mainFiles) > 0 {
			return exec.Command("powershell", "-Command", "go mod download && go run "+mainFiles[0]), nil
		}
		// Fallback to go run .
		return exec.Command("powershell", "-Command", "go mod download && go run ."), nil
	}

	// Check for Node.js project
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		// Check if there's a dev script, otherwise use start
		devCommand := getNpmDevCommand(projectPath)
		return exec.Command("powershell", "-Command", "npm install && "+devCommand), nil
	}

	// Check for Python project
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		// Check for Flask app.py or Django manage.py
		devCommand := getPythonDevCommand(projectPath)
		return exec.Command("powershell", "-Command", "pip install -r requirements.txt && "+devCommand), nil
	}

	// Check for Rust project
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		return exec.Command("powershell", "-Command", "cargo build && cargo run"), nil
	}

	// Check for .NET project
	if matches, _ := filepath.Glob(filepath.Join(projectPath, "*.csproj")); len(matches) > 0 {
		return exec.Command("powershell", "-Command", "dotnet restore && dotnet watch run"), nil
	}

	// Check for Java Maven project
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		return exec.Command("powershell", "-Command", "mvn dependency:resolve && mvn exec:java"), nil
	}

	// Check for Java Gradle project
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); err == nil {
		return exec.Command("powershell", "-Command", "./gradlew build && ./gradlew run"), nil
	}

	return nil, fmt.Errorf("unable to detect project type or run command")
}

// scanProjectsWithPathCmd creates a command that scans for projects at a specific path
func scanProjectsWithPathCmd(scanPath string) tea.Cmd {
	return func() tea.Msg {
		// Scan for projects at the specified path
		projects, err := engine.ScanDirectory(scanPath)
		if err != nil {
			return ScanCompleteMsg{err: err}
		}

		// Get existing projects from database
		existingProjects, err := db.GetProjects()
		if err != nil {
			return ScanCompleteMsg{err: err}
		}

		// Create map of scanned project paths
		scannedPaths := make(map[string]bool)
		for _, p := range projects {
			scannedPaths[p.Path] = true
		}

		// Remove projects that no longer exist (only active ones)
		removedCount := 0
		for _, existing := range existingProjects {
			if existing.Status == "active" && !scannedPaths[existing.Path] {
				if err := db.DeleteProject(existing.ID); err == nil {
					removedCount++
				}
			}
		}

		// Add new projects to database
		addedCount := 0
		for i := range projects {
			if err := db.AddProject(&projects[i]); err == nil {
				addedCount++
			}
		}

		return ScanCompleteMsg{
			projectsFound:   len(projects),
			projectsAdded:   addedCount,
			projectsRemoved: removedCount,
		}
	}
}

// reloadProjectsCmd creates a command that reloads the project list
func reloadProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		projects, err := db.GetProjects()
		if err != nil {
			return ErrorMsg{err: err}
		}

		// Convert to list items
		items := make([]list.Item, len(projects))
		for i, p := range projects {
			items[i] = projectItem{project: p, isLoading: false}
		}

		return reloadMsg{items: items}
	}
}

// reloadMsg is sent when the project list needs to be reloaded
type reloadMsg struct {
	items []list.Item
}

// clearAllProjectsCmd creates a command that clears all projects from the database
func clearAllProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		count, err := db.DeleteAllProjects()
		return ClearAllMsg{
			count: count,
			err:   err,
		}
	}
}

// cloneProjectCmd creates a command that clones a GitHub repository and adds it to the database
func cloneProjectCmd(repoURL, rootPath string) tea.Cmd {
	return func() tea.Msg {
		// Parse repo name from URL
		// Expected format: https://github.com/owner/repo or https://github.com/owner/repo.git
		parts := strings.Split(repoURL, "/")
		if len(parts) < 2 {
			return CloneMsg{err: fmt.Errorf("invalid GitHub URL format")}
		}
		repoName := parts[len(parts)-1]
		// Remove .git suffix if present
		repoName = strings.TrimSuffix(repoName, ".git")

		// Determine project path
		projectPath := filepath.Join(rootPath, repoName)

		// Check if project already exists
		if _, err := db.GetProjectByPath(projectPath); err == nil {
			return CloneMsg{err: fmt.Errorf("project already exists at %s", projectPath)}
		}

		// Clone the repository
		if err := engine.CloneRepository(repoURL, projectPath); err != nil {
			return CloneMsg{err: err}
		}

		// Create project record
		project := &models.Project{
			Name:    repoName,
			Path:    projectPath,
			RepoURL: repoURL,
			Status:  "active",
		}

		// Add to database
		if err := db.AddProject(project); err != nil {
			// Clean up cloned directory on failure
			os.RemoveAll(projectPath)
			return CloneMsg{err: err}
		}

		return CloneMsg{
			projectName: repoName,
			projectPath: projectPath,
		}
	}
}

// syncToCloudCmd creates a command that syncs projects to GitHub Gist
func syncToCloudCmd() tea.Cmd {
	return func() tea.Msg {
		// Get GitHub token from config
		token, err := db.GetConfig("github_token")
		if err != nil || token == "" {
			return SyncToCloudMsg{err: fmt.Errorf("GitHub authentication required. Please authenticate with OAuth (press 't')")}
		}

		// Validate token
		validationClient := engine.NewGistClient(token)
		if err := validationClient.ValidateToken(); err != nil {
			return SyncToCloudMsg{err: fmt.Errorf("invalid GitHub token. Please reconfigure your token (press 't')")}
		}

		// Get existing gist ID from config
		gistID, _ := db.GetConfig("gist_id")

		// Get all projects
		projects, err := db.GetProjects()
		if err != nil {
			return SyncToCloudMsg{err: fmt.Errorf("failed to get projects: %w", err)}
		}

		// Create gist client
		client := engine.NewGistClient(token)

		// Save to gist
		newGistID, err := client.SaveToGist(projects, gistID)
		if err != nil {
			return SyncToCloudMsg{err: err}
		}

		return SyncToCloudMsg{gistID: newGistID}
	}
}

// loadFromCloudCmd creates a command that loads projects from GitHub Gist
func loadFromCloudCmd() tea.Cmd {
	return func() tea.Msg {
		// Get GitHub token from config
		token, err := db.GetConfig("github_token")
		if err != nil || token == "" {
			return LoadFromCloudMsg{err: fmt.Errorf("GitHub authentication required. Please authenticate with OAuth (press 't')")}
		}

		// Validate token
		validationClient := engine.NewGistClient(token)
		if err := validationClient.ValidateToken(); err != nil {
			return LoadFromCloudMsg{err: fmt.Errorf("invalid GitHub token. Please reconfigure your token (press 't')")}
		}

		// Get gist ID from config
		gistID, err := db.GetConfig("gist_id")
		if err != nil || gistID == "" {
			return LoadFromCloudMsg{err: fmt.Errorf("gist ID not configured. Please sync to cloud first")}
		}

		// Create gist client
		client := engine.NewGistClient(token)

		// Load from gist
		projects, err := client.LoadFromGist(gistID)
		if err != nil {
			return LoadFromCloudMsg{err: err}
		}

		// Clear existing projects
		if _, err := db.DeleteAllProjects(); err != nil {
			return LoadFromCloudMsg{err: fmt.Errorf("failed to clear existing projects: %w", err)}
		}

		// Add loaded projects
		for _, project := range projects {
			project.ID = 0 // Reset ID for new insertion
			if err := db.AddProject(&project); err != nil {
				return LoadFromCloudMsg{err: fmt.Errorf("failed to add project %s: %w", project.Name, err)}
			}
		}

		return LoadFromCloudMsg{projectsLoaded: len(projects)}
	}
}

// listCloudProjectsCmd creates a command that lists projects from GitHub Gist
func listCloudProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		// Get GitHub token from config
		token, err := db.GetConfig("github_token")
		if err != nil || token == "" {
			return ListCloudProjectsMsg{err: fmt.Errorf("GitHub authentication required. Please authenticate with OAuth (press 't')")}
		}

		// Validate token
		validationClient := engine.NewGistClient(token)
		if err := validationClient.ValidateToken(); err != nil {
			return ListCloudProjectsMsg{err: fmt.Errorf("invalid GitHub token")}
		}

		// Get gist ID from config
		gistID, err := db.GetConfig("gist_id")
		if err != nil || gistID == "" {
			return ListCloudProjectsMsg{err: fmt.Errorf("no cloud backup found. Please sync to cloud first")}
		}

		// Create gist client
		client := engine.NewGistClient(token)

		// Load projects from gist
		projects, err := client.ListProjectsFromGist(gistID)
		if err != nil {
			return ListCloudProjectsMsg{err: err}
		}

		return ListCloudProjectsMsg{projects: projects}
	}
}

// loadSelectedProjectsCmd creates a command that loads selected projects from cloud
func loadSelectedProjectsCmd(selectedIndices []int, cloudProjects []models.Project) tea.Cmd {
	return func() tea.Msg {
		loadedCount := 0

		for _, idx := range selectedIndices {
			if idx < 0 || idx >= len(cloudProjects) {
				continue
			}

			project := cloudProjects[idx]
			// Reset ID for new insertion and mark as archived
			project.ID = 0
			project.Status = "archived"

			// Check if project already exists
			if existing, err := db.GetProjectByPath(project.Path); err == nil {
				// Update existing project
				project.ID = existing.ID
				if err := db.UpdateProject(&project); err != nil {
					continue
				}
			} else {
				// Add new project
				if err := db.AddProject(&project); err != nil {
					continue
				}
			}
			loadedCount++
		}

		return LoadSelectedProjectsMsg{projectsLoaded: loadedCount}
	}
}

// initiateOAuthCmd creates a command that initiates the GitHub OAuth device flow
func initiateOAuthCmd() tea.Cmd {
	return func() tea.Msg {
		oauthClient := engine.NewOAuthClient()

		deviceResp, err := oauthClient.InitiateDeviceFlow()
		if err != nil {
			return OAuthDeviceCodeMsg{err: err}
		}

		return OAuthDeviceCodeMsg{
			deviceCode:      deviceResp.DeviceCode,
			userCode:        deviceResp.UserCode,
			verificationURI: deviceResp.VerificationURI,
			interval:        deviceResp.Interval,
			err:             nil,
		}
	}
}

// pollForAccessTokenCmd creates a command that polls for the OAuth access token
func pollForAccessTokenCmd(deviceCode string, interval int) tea.Cmd {
	return func() tea.Msg {
		oauthClient := engine.NewOAuthClient()

		accessToken, err := oauthClient.PollForAccessToken(deviceCode, interval)
		if err != nil {
			return OAuthCompleteMsg{err: err}
		}

		return OAuthCompleteMsg{
			accessToken: accessToken,
			err:         nil,
		}
	}
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getFilteredIndices returns a list of original indices that match the filter
func (m model) getFilteredIndices() []int {
	filterText := strings.ToLower(strings.TrimSpace(m.cloudFilterInput.Value()))
	if filterText == "" {
		// No filter - return all indices
		indices := make([]int, len(m.cloudProjects))
		for i := range m.cloudProjects {
			indices[i] = i
		}
		return indices
	}

	// Filter and return matching indices
	indices := []int{}
	for i, project := range m.cloudProjects {
		if strings.Contains(strings.ToLower(project.Name), filterText) ||
			strings.Contains(strings.ToLower(project.Path), filterText) ||
			strings.Contains(strings.ToLower(project.RepoURL), filterText) {
			indices = append(indices, i)
		}
	}
	return indices
}
