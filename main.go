package main

import (
	"devbase/db"
	"devbase/models"
	"fmt"
	"log"
	"time"
)

func main() {
	// Initialize the database
	fmt.Println("Initializing DevBase database...")
	if err := db.InitDB("devbase.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	fmt.Println("✓ Database initialized successfully with WAL mode\n")

	// Test 1: Add some sample projects
	fmt.Println("=== Test 1: Adding sample projects ===")
	
	project1 := &models.Project{
		Name:       "DevBase",
		Path:       "D:\\WigerLabs\\Projects\\CLI_tools\\DevBase",
		RepoURL:    "https://github.com/example/devbase",
		Status:     "active",
		LastOpened: time.Now(),
		Tags:       []string{"go", "cli", "database"},
	}

	project2 := &models.Project{
		Name:       "WebApp",
		Path:       "D:\\Projects\\WebApp",
		RepoURL:    "https://github.com/example/webapp",
		Status:     "active",
		LastOpened: time.Now().Add(-24 * time.Hour),
		Tags:       []string{"react", "typescript", "frontend"},
	}

	project3 := &models.Project{
		Name:       "OldProject",
		Path:       "D:\\Projects\\OldProject",
		RepoURL:    "",
		Status:     "archived",
		LastOpened: time.Now().Add(-7 * 24 * time.Hour),
		Tags:       []string{"legacy", "archived"},
	}

	if err := db.AddProject(project1); err != nil {
		log.Printf("Error adding project1: %v", err)
	} else {
		fmt.Printf("✓ Added project: %s (ID: %d)\n", project1.Name, project1.ID)
	}

	if err := db.AddProject(project2); err != nil {
		log.Printf("Error adding project2: %v", err)
	} else {
		fmt.Printf("✓ Added project: %s (ID: %d)\n", project2.Name, project2.ID)
	}

	if err := db.AddProject(project3); err != nil {
		log.Printf("Error adding project3: %v", err)
	} else {
		fmt.Printf("✓ Added project: %s (ID: %d)\n", project3.Name, project3.ID)
	}

	// Test 2: Retrieve all projects sorted by LastOpened
	fmt.Println("\n=== Test 2: Retrieving all projects (sorted by LastOpened DESC) ===")
	projects, err := db.GetProjects()
	if err != nil {
		log.Fatalf("Failed to retrieve projects: %v", err)
	}

	for i, p := range projects {
		fmt.Printf("\n%d. %s\n", i+1, p.Name)
		fmt.Printf("   ID: %d\n", p.ID)
		fmt.Printf("   Path: %s\n", p.Path)
		fmt.Printf("   RepoURL: %s\n", p.RepoURL)
		fmt.Printf("   Status: %s\n", p.Status)
		fmt.Printf("   LastOpened: %s\n", p.LastOpened.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Tags: %v\n", p.Tags)
	}

	// Test 3: Get project by ID
	fmt.Println("\n=== Test 3: Get project by ID ===")
	project, err := db.GetProjectByID(1)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Found project ID 1: %s\n", project.Name)
	}

	// Test 4: Get project by Path
	fmt.Println("\n=== Test 4: Get project by Path ===")
	project, err = db.GetProjectByPath("D:\\Projects\\WebApp")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Found project at path: %s\n", project.Name)
	}

	// Test 5: Update LastOpened
	fmt.Println("\n=== Test 5: Update LastOpened timestamp ===")
	oldTime := project2.LastOpened
	time.Sleep(100 * time.Millisecond)
	if err := db.UpdateLastOpened(project2.ID); err != nil {
		log.Printf("Error: %v", err)
	} else {
		updated, _ := db.GetProjectByID(project2.ID)
		fmt.Printf("✓ Updated LastOpened for %s\n", updated.Name)
		fmt.Printf("  Old: %s\n", oldTime.Format("2006-01-02 15:04:05.000"))
		fmt.Printf("  New: %s\n", updated.LastOpened.Format("2006-01-02 15:04:05.000"))
	}

	// Test 6: Update project
	fmt.Println("\n=== Test 6: Update project ===")
	project1.Tags = append(project1.Tags, "sqlite", "gorm")
	project1.Status = "archived"
	if err := db.UpdateProject(project1); err != nil {
		log.Printf("Error: %v", err)
	} else {
		updated, _ := db.GetProjectByID(project1.ID)
		fmt.Printf("✓ Updated project: %s\n", updated.Name)
		fmt.Printf("  Status: %s\n", updated.Status)
		fmt.Printf("  Tags: %v\n", updated.Tags)
	}

	// Test 7: Validate status check
	fmt.Println("\n=== Test 7: Test status validation ===")
	invalidProject := &models.Project{
		Name:   "Invalid",
		Path:   "D:\\Invalid",
		Status: "unknown",
	}
	if err := db.AddProject(invalidProject); err != nil {
		fmt.Printf("✓ Status validation working: %v\n", err)
	}

	fmt.Println("\n=== All tests completed successfully! ===")
	fmt.Printf("\nTotal projects in database: %d\n", len(projects))
}
