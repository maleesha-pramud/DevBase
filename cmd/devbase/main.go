package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"devbase/db"
	"devbase/ui"
)

const version = "1.0.0"

func main() {
	// Check for command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("DevBase v%s\n", version)
			return
		case "--help", "-h":
			printHelp()
			return
		case "scan":
			handleScan()
			return
		}
	}

	// Initialize the database with proper path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	dbPath := filepath.Join(homeDir, "devbase.db")

	if err := db.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	// Check if database is empty
	projects, err := db.GetProjects()
	if err != nil {
		log.Fatalf("Failed to check projects: %v", err)
	}

	if len(projects) == 0 {
		fmt.Println("╔═══════════════════════════════════════════════════════════╗")
		fmt.Println("║              Welcome to DevBase v" + version + "                   ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════╝")
		fmt.Println("\nNo projects found in database.")
		fmt.Println("\nOptions:")
		fmt.Println("  1. Run 'DevBase scan' to scan your directories for projects")
		fmt.Println("  2. Add projects manually using the UI (press 'a' in the app)")
		fmt.Println("\nStarting empty UI...")
		fmt.Println()
	}

	// Create and run the Bubble Tea UI
	m, err := ui.NewModel()
	if err != nil {
		log.Fatalf("Failed to create UI model: %v", err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

func printHelp() {
	fmt.Printf(`DevBase v%s - Project Manager CLI Tool

USAGE:
    DevBase [command]

COMMANDS:
    scan            Scan directories for projects and add them to database
    --help, -h      Show this help message
    --version, -v   Show version information

INTERACTIVE MODE (default):
    When no command is provided, DevBase starts in interactive mode.

KEYBOARD SHORTCUTS:
    enter           Open project in VS Code
    s               Scan for new projects
    x               Run project in development mode
    d               Archive selected project (deletes directory)
    r               Restore archived project (clones from repo)
    f               Manage root folders (press 'e' there to execute commands)
    /               Filter/search projects
    q, ctrl+c       Quit

FEATURES:
    • Optimistic UI updates for instant feedback
    • Automatic project discovery (Go, Node.js, Git repos)
    • SQLite database with WAL mode for performance
    • Shallow git cloning for fast project restoration
    • Concurrent directory scanning with worker pools

REQUIREMENTS:
    • VS Code installed with 'code' command in PATH
    • Git installed (for restore functionality)

For more information, visit: github.com/example/devbase
`, version)
}

func handleScan() {
	fmt.Println("Scan functionality will be added via the UI.")
	fmt.Println("Please use interactive mode and press 's' to scan.")
	os.Exit(1)
}
