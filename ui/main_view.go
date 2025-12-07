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
	screenSetup screenState = iota
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
	screen              screenState
	pathInput           textinput.Model
	list                list.Model
	errorMessage        string
	statusMessage       string
	isScanning          bool
	confirmClearAll     bool
	confirmArchive      bool
	archiveConfirmInput textinput.Model
	archiveProject      *projectItem
	archiveIdx          int
	confirmClone        bool
	cloneInput          textinput.Model
	rootScanPath        string
	width               int
	height              int
	ready               bool
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
	if m.screen == screenSetup {
		return m.updateSetup(msg)
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
			if m.screen == screenSetup {
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
			m.screen = screenSetup

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
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			// Start scanning with the entered path
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
		// Switch to list screen
		m.screen = screenList
		return m, reloadProjectsCmd()

	case reloadMsg:
		// Load projects into list and switch to list screen
		m.list.SetItems(msg.items)
		m.screen = screenList
		return m, nil
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

// View renders the UI
func (m model) View() string {
	if m.screen == screenSetup {
		return m.viewSetup()
	}
	return m.viewList()
}

// viewSetup renders the setup screen
func (m model) viewSetup() string {
	var s string

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

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("\n\nPress Enter to start scan | Ctrl+C or Q to quit")
	s += helpText

	return docStyle.Render(s)
}

// viewList renders the project list screen
func (m model) viewList() string {
	// If not ready, show loading state
	if !m.ready {
		return "Loading..."
	}

	view := m.list.View()

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

		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

		restoreInfoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00"))

		noRestoreStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00"))

		archivePrompt = "\n\n" +
			titleStyle.Render("⚠ WARNING: ARCHIVE PROJECT") + "\n\n" +
			infoStyle.Render(fmt.Sprintf("Project: %s\n", m.archiveProject.project.Name)) +
			infoStyle.Render(fmt.Sprintf("Path: %s\n\n", m.archiveProject.project.Path))

		if hasRepoURL {
			archivePrompt += restoreInfoStyle.Render("✓ This project CAN be restored from:\n") +
				restoreInfoStyle.Render(fmt.Sprintf("  %s\n\n", m.archiveProject.project.RepoURL))
		} else {
			archivePrompt += noRestoreStyle.Render("⚠ WARNING: No git repository URL found!\n") +
				noRestoreStyle.Render("  This project CANNOT be restored after archiving.\n\n")
		}

		archivePrompt += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			Render("Type 'DELETE' to confirm archive: ") + "\n" +
			m.archiveConfirmInput.View() + "\n\n" +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Render("Press Enter to confirm | ESC to cancel")
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
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("\n\nKeys: enter=open  s=scan  g=clone  c=clear-all  d=archive  r=restore  /=filter  q=quit")

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

		return model{
			screen:          screenSetup,
			pathInput:       ti,
			list:            l,
			errorMessage:    "",
			statusMessage:   "",
			isScanning:      false,
			confirmClearAll: false,
			confirmArchive:  false,
			confirmClone:    false,
			cloneInput:      textinput.New(),
			rootScanPath:    rootPath,
			width:           80,
			height:          24,
			ready:           false,
		}, nil
	}

	// Convert projects to list items
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p, isLoading: false}
	}
	l.SetItems(items)

	return model{
		screen:          screenList,
		pathInput:       textinput.New(),
		list:            l,
		errorMessage:    "",
		statusMessage:   "",
		isScanning:      false,
		confirmClearAll: false,
		confirmArchive:  false,
		confirmClone:    false,
		cloneInput:      textinput.New(),
		rootScanPath:    rootPath,
		width:           80,
		height:          24,
		ready:           false,
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
