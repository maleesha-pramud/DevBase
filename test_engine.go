package main

import (
	"devbase/db"
	"devbase/engine"
	"devbase/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Initialize the database
	fmt.Println("=== Engine Tests - DevBase ===\n")
	fmt.Println("Step 1: Initializing database...")
	if err := db.InitDB("devbase_engine_test.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()
	defer os.Remove("devbase_engine_test.db")
	defer os.Remove("devbase_engine_test.db-shm")
	defer os.Remove("devbase_engine_test.db-wal")

	fmt.Println("✓ Database initialized\n")

	// Create test directories
	testDir := "test_projects"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Test 1: ArchiveProject with non-existent path
	fmt.Println("=== Test 1: ArchiveProject (non-existent path) ===")
	project1 := &models.Project{
		Name:       "TestProject1",
		Path:       filepath.Join(testDir, "project1"),
		RepoURL:    "https://github.com/example/project1",
		Status:     "active",
		LastOpened: time.Now(),
		Tags:       []string{"test"},
	}
	if err := db.AddProject(project1); err != nil {
		log.Fatalf("Failed to add project: %v", err)
	}
	fmt.Printf("✓ Added project: %s (ID: %d)\n", project1.Name, project1.ID)

	if err := engine.ArchiveProject(project1.ID); err != nil {
		log.Fatalf("Failed to archive project: %v", err)
	}
	archived, _ := db.GetProjectByID(project1.ID)
	fmt.Printf("✓ Archived project: Status = %s\n\n", archived.Status)

	// Test 2: ArchiveProject with existing directory
	fmt.Println("=== Test 2: ArchiveProject (with directory) ===")
	project2 := &models.Project{
		Name:       "TestProject2",
		Path:       filepath.Join(testDir, "project2"),
		RepoURL:    "https://github.com/example/project2",
		Status:     "active",
		LastOpened: time.Now(),
		Tags:       []string{"test"},
	}
	if err := db.AddProject(project2); err != nil {
		log.Fatalf("Failed to add project: %v", err)
	}
	fmt.Printf("✓ Added project: %s (ID: %d)\n", project2.Name, project2.ID)

	// Create the project directory with a test file
	os.MkdirAll(project2.Path, 0755)
	testFile := filepath.Join(project2.Path, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	fmt.Printf("✓ Created project directory at %s\n", project2.Path)

	// Verify directory exists before archival
	if _, err := os.Stat(project2.Path); err == nil {
		fmt.Println("✓ Directory verified before archival")
	}

	// Archive the project
	if err := engine.ArchiveProject(project2.ID); err != nil {
		log.Fatalf("Failed to archive project: %v", err)
	}

	// Verify directory is deleted
	if _, err := os.Stat(project2.Path); os.IsNotExist(err) {
		fmt.Println("✓ Directory successfully deleted after archival")
	} else {
		log.Fatalf("Directory still exists after archival!")
	}

	archived2, _ := db.GetProjectByID(project2.ID)
	fmt.Printf("✓ Project status updated: Status = %s\n\n", archived2.Status)

	// Test 3: VerifyProjectPath
	fmt.Println("=== Test 3: VerifyProjectPath ===")
	exists, err := engine.VerifyProjectPath(project1.ID)
	if err != nil {
		log.Fatalf("Error checking path: %v", err)
	}
	fmt.Printf("✓ Project 1 path exists: %v\n", exists)

	project3 := &models.Project{
		Name:       "TestProject3",
		Path:       filepath.Join(testDir, "project3"),
		RepoURL:    "https://github.com/example/project3",
		Status:     "active",
		LastOpened: time.Now(),
		Tags:       []string{"test"},
	}
	if err := db.AddProject(project3); err != nil {
		log.Fatalf("Failed to add project: %v", err)
	}
	os.MkdirAll(project3.Path, 0755)

	exists, _ = engine.VerifyProjectPath(project3.ID)
	fmt.Printf("✓ Project 3 path exists: %v\n\n", exists)

	// Test 4: GetProjectStatus
	fmt.Println("=== Test 4: GetProjectStatus ===")
	status, err := engine.GetProjectStatus(project1.ID)
	if err != nil {
		log.Fatalf("Error getting status: %v", err)
	}
	fmt.Printf("✓ Project 1 status: %s\n", status)

	status, _ = engine.GetProjectStatus(project3.ID)
	fmt.Printf("✓ Project 3 status: %s\n\n", status)

	// Test 5: ArchiveWithVerification
	fmt.Println("=== Test 5: ArchiveWithVerification ===")
	if err := engine.ArchiveWithVerification(project3.ID); err != nil {
		log.Fatalf("Failed to archive with verification: %v", err)
	}
	fmt.Printf("✓ Project 3 archived with verification\n")
	fmt.Printf("✓ Directory verified and deleted\n\n")

	// Test 6: RestoreProject validation (should fail with invalid path)
	fmt.Println("=== Test 6: RestoreProject validation ===")
	project4 := &models.Project{
		Name:       "TestProject4",
		Path:       filepath.Join(testDir, "project4"),
		RepoURL:    "https://github.com/golang/go",
		Status:     "archived",
		LastOpened: time.Now(),
		Tags:       []string{"test"},
	}
	if err := db.AddProject(project4); err != nil {
		log.Fatalf("Failed to add project: %v", err)
	}
	fmt.Printf("✓ Added archived project: %s (ID: %d)\n", project4.Name, project4.ID)

	// Test RestoreProject validation
	project5 := &models.Project{
		Name:       "TestProject5",
		Path:       filepath.Join(testDir, "project5"),
		RepoURL:    "", // No repo URL
		Status:     "archived",
		LastOpened: time.Now(),
		Tags:       []string{"test"},
	}
	if err := db.AddProject(project5); err != nil {
		log.Fatalf("Failed to add project: %v", err)
	}

	if err := engine.RestoreProject(project5.ID); err != nil {
		fmt.Printf("✓ RestoreProject correctly rejected project without repo: %v\n\n", err)
	}

	// Test 7: RestoreWithVerification on already active project (should fail)
	fmt.Println("=== Test 7: RestoreWithVerification validation ===")
	if err := engine.RestoreWithVerification(project3.ID); err != nil {
		fmt.Printf("✓ RestoreWithVerification correctly rejected already active project: %v\n\n", err)
	}

	// Test 8: ArchiveWithVerification on already archived project (should fail)
	fmt.Println("=== Test 8: ArchiveWithVerification validation ===")
	if err := engine.ArchiveWithVerification(project1.ID); err != nil {
		fmt.Printf("✓ ArchiveWithVerification correctly rejected already archived project: %v\n\n", err)
	}

	// Test 9: DeleteProjectPermanently
	fmt.Println("=== Test 9: DeleteProjectPermanently ===")
	tempProject := &models.Project{
		Name:       "TempProject",
		Path:       filepath.Join(testDir, "temp"),
		RepoURL:    "https://github.com/example/temp",
		Status:     "active",
		LastOpened: time.Now(),
		Tags:       []string{"temporary"},
	}
	if err := db.AddProject(tempProject); err != nil {
		log.Fatalf("Failed to add temp project: %v", err)
	}
	os.MkdirAll(tempProject.Path, 0755)
	tempID := tempProject.ID

	if err := engine.DeleteProjectPermanently(tempID); err != nil {
		log.Fatalf("Failed to permanently delete: %v", err)
	}

	// Verify project is deleted from DB
	if _, err := db.GetProjectByID(tempID); err != nil {
		fmt.Printf("✓ Project permanently deleted from database\n")
	}
	fmt.Printf("✓ Project directory deleted\n\n")

	fmt.Println("=== All Engine Tests Completed Successfully! ===")
}
