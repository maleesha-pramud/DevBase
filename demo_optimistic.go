package main

import (
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"devbase/db"
	"devbase/models"
	"devbase/ui"
)

func main() {
	fmt.Println("=== DevBase with Optimistic UI Updates ===\n")

	// Initialize the database
	if err := db.InitDB("devbase.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	// Check if we need to seed some sample data
	projects, err := db.GetProjects()
	if err != nil {
		log.Fatalf("Failed to check projects: %v", err)
	}

	// Seed sample data if database is empty
	if len(projects) == 0 {
		fmt.Println("Database is empty. Seeding sample data...")
		seedSampleData()
		fmt.Println("Sample data added!\n")
	}

	fmt.Println("Starting DevBase UI...")
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              OPTIMISTIC UI DEMONSTRATION                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println("\nFeatures:")
	fmt.Println("  • Press 'd' to archive a project (IMMEDIATE UI update)")
	fmt.Println("  • Press 'r' to restore an archived project")
	fmt.Println("  • UI updates INSTANTLY, DB operations happen in background")
	fmt.Println("  • If operation fails, UI automatically rolls back")
	fmt.Println("  • Press '/' to filter/search projects")
	fmt.Println("  • Press 'q' to quit")
	fmt.Println("\nNote: Archive will delete the directory if it exists.")
	fmt.Println("      Restore will attempt to clone from RepoURL.")
	fmt.Println("\nStarting in 2 seconds...\n")
	time.Sleep(2 * time.Second)

	// Create the Bubble Tea model
	m, err := ui.NewModel()
	if err != nil {
		log.Fatalf("Failed to create UI model: %v", err)
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}

	fmt.Println("\nThank you for using DevBase!")
}

func seedSampleData() {
	sampleProjects := []models.Project{
		{
			Name:       "DevBase",
			Path:       "D:\\WigerLabs\\Projects\\CLI_tools\\DevBase",
			RepoURL:    "https://github.com/example/devbase",
			Status:     "active",
			LastOpened: time.Now(),
			Tags:       []string{"go", "cli", "database", "bubbletea"},
		},
		{
			Name:       "WebApp",
			Path:       "D:\\Projects\\WebApp",
			RepoURL:    "https://github.com/example/webapp",
			Status:     "active",
			LastOpened: time.Now().Add(-24 * time.Hour),
			Tags:       []string{"react", "typescript", "frontend"},
		},
		{
			Name:       "MobileApp",
			Path:       "D:\\Projects\\MobileApp",
			RepoURL:    "https://github.com/example/mobile",
			Status:     "active",
			LastOpened: time.Now().Add(-48 * time.Hour),
			Tags:       []string{"flutter", "mobile"},
		},
		{
			Name:       "APIService",
			Path:       "D:\\Work\\APIService",
			RepoURL:    "https://github.com/company/api",
			Status:     "active",
			LastOpened: time.Now().Add(-72 * time.Hour),
			Tags:       []string{"go", "api", "backend"},
		},
		{
			Name:       "DataPipeline",
			Path:       "D:\\Projects\\DataPipeline",
			RepoURL:    "https://github.com/example/pipeline",
			Status:     "active",
			LastOpened: time.Now().Add(-96 * time.Hour),
			Tags:       []string{"python", "data", "etl"},
		},
		{
			Name:       "OldProject",
			Path:       "D:\\Archive\\OldProject",
			RepoURL:    "https://github.com/example/old",
			Status:     "archived",
			LastOpened: time.Now().Add(-30 * 24 * time.Hour),
			Tags:       []string{"legacy", "archived"},
		},
	}

	for _, p := range sampleProjects {
		proj := p // Create a copy
		if err := db.AddProject(&proj); err != nil {
			fmt.Printf("Warning: Could not add %s: %v\n", p.Name, err)
		}
	}
}
