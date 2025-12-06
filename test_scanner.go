package main

import (
	"devbase/db"
	"devbase/engine"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Initialize the database
	fmt.Println("=== Scanner Tests - DevBase ===\n")
	fmt.Println("Step 1: Initializing database...")
	if err := db.InitDB("devbase_scanner_test.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()
	defer os.Remove("devbase_scanner_test.db")
	defer os.Remove("devbase_scanner_test.db-shm")
	defer os.Remove("devbase_scanner_test.db-wal")

	fmt.Println("✓ Database initialized\n")

	// Create test directory structure
	testRoot := "test_scan_root"
	os.RemoveAll(testRoot) // Clean up any previous test
	defer os.RemoveAll(testRoot)

	fmt.Println("Step 2: Creating test directory structure...")
	createTestStructure(testRoot)
	fmt.Println("✓ Test structure created\n")

	// Test 1: Scan directory
	fmt.Println("=== Test 1: Scan Directory for Projects ===")
	startTime := time.Now()
	projects, err := engine.ScanDirectory(testRoot)
	if err != nil {
		log.Fatalf("Failed to scan directory: %v", err)
	}
	elapsed := time.Since(startTime)

	fmt.Printf("✓ Scan completed in %v\n", elapsed)
	fmt.Printf("✓ Found %d projects\n\n", len(projects))

	// Display found projects
	fmt.Println("=== Found Projects ===")
	for i, p := range projects {
		fmt.Printf("%d. %s\n", i+1, p.Name)
		fmt.Printf("   Path: %s\n", p.Path)
		fmt.Printf("   Status: %s\n\n", p.Status)
	}

	// Verify expected projects were found
	expectedProjects := map[string]bool{
		"go_project":   false,
		"node_project": false,
		"git_project":  false,
	}

	for _, p := range projects {
		if _, exists := expectedProjects[p.Name]; exists {
			expectedProjects[p.Name] = true
		}
	}

	fmt.Println("=== Verification ===")
	allFound := true
	for name, found := range expectedProjects {
		status := "✓"
		if !found {
			status = "✗"
			allFound = false
		}
		fmt.Printf("%s Expected project '%s' found: %v\n", status, name, found)
	}

	// Verify ignored directories were not scanned
	fmt.Println("\n=== Ignored Directory Verification ===")
	ignoredFound := false
	for _, p := range projects {
		if p.Name == "ignored_project" {
			ignoredFound = true
			break
		}
	}
	if !ignoredFound {
		fmt.Println("✓ node_modules correctly ignored")
	} else {
		fmt.Println("✗ node_modules was NOT ignored (error)")
		allFound = false
	}

	// Test 2: Add scanned projects to database
	fmt.Println("\n=== Test 2: Add Scanned Projects to Database ===")
	addedCount := 0
	for i := range projects {
		// Ensure LastOpened is set
		if projects[i].LastOpened.IsZero() {
			projects[i].LastOpened = time.Now()
		}
		if err := db.AddProject(&projects[i]); err != nil {
			fmt.Printf("Warning: Failed to add project %s: %v\n", projects[i].Name, err)
		} else {
			addedCount++
		}
	}
	fmt.Printf("✓ Added %d projects to database\n", addedCount)

	// Verify database contents
	dbProjects, err := db.GetProjects()
	if err != nil {
		log.Fatalf("Failed to retrieve projects from database: %v", err)
	}
	fmt.Printf("✓ Database now contains %d projects\n", len(dbProjects))

	if allFound {
		fmt.Println("\n=== All Scanner Tests Passed! ===")
	} else {
		fmt.Println("\n=== Some Scanner Tests Failed ===")
		os.Exit(1)
	}
}

func createTestStructure(root string) {
	// Create Go project
	goProject := filepath.Join(root, "projects", "go_project")
	os.MkdirAll(goProject, 0755)
	os.WriteFile(filepath.Join(goProject, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(goProject, "main.go"), []byte("package main"), 0644)

	// Create Node.js project
	nodeProject := filepath.Join(root, "projects", "node_project")
	os.MkdirAll(nodeProject, 0755)
	os.WriteFile(filepath.Join(nodeProject, "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(nodeProject, "index.js"), []byte("console.log('test')"), 0644)

	// Create Git project (without go.mod or package.json)
	gitProject := filepath.Join(root, "projects", "git_project")
	gitDir := filepath.Join(gitProject, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("[core]"), 0644)
	os.WriteFile(filepath.Join(gitProject, "README.md"), []byte("# Test"), 0644)

	// Create a project inside node_modules (should be ignored)
	nodeModules := filepath.Join(root, "projects", "node_project", "node_modules", "ignored_project")
	os.MkdirAll(nodeModules, 0755)
	os.WriteFile(filepath.Join(nodeModules, "package.json"), []byte("{}"), 0644)

	// Create a non-project directory
	nonProject := filepath.Join(root, "documents")
	os.MkdirAll(nonProject, 0755)
	os.WriteFile(filepath.Join(nonProject, "readme.txt"), []byte("not a project"), 0644)

	// Create nested projects
	nestedGo := filepath.Join(root, "work", "clients", "acme", "backend")
	os.MkdirAll(nestedGo, 0755)
	os.WriteFile(filepath.Join(nestedGo, "go.mod"), []byte("module acme"), 0644)

	// Create project in dist folder (should be ignored)
	distProject := filepath.Join(root, "projects", "node_project", "dist", "ignored_dist")
	os.MkdirAll(distProject, 0755)
	os.WriteFile(filepath.Join(distProject, "package.json"), []byte("{}"), 0644)

	// Create project in build folder (should be ignored)
	buildProject := filepath.Join(root, "projects", "go_project", "build", "ignored_build")
	os.MkdirAll(buildProject, 0755)
	os.WriteFile(filepath.Join(buildProject, "go.mod"), []byte("module build"), 0644)

	// Create project in vendor folder (should be ignored)
	vendorProject := filepath.Join(root, "projects", "go_project", "vendor", "github.com", "user", "ignored_vendor")
	os.MkdirAll(vendorProject, 0755)
	os.WriteFile(filepath.Join(vendorProject, "go.mod"), []byte("module vendor"), 0644)
}
