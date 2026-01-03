package db

import (
	"devbase/models"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestDB initializes a test database in a temporary location
func setupTestDB(t *testing.T) string {
	// Create a temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize the database
	err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return dbPath
}

// teardownTestDB closes the database connection
func teardownTestDB(t *testing.T) {
	if err := CloseDB(); err != nil {
		t.Errorf("Failed to close test database: %v", err)
	}
}

// TestRootFolderCRUD tests Create, Read, Update, Delete operations for RootFolder
func TestRootFolderCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// Test 1: Add a root folder
	rootFolder := &models.RootFolder{
		Name:     "Test Projects",
		Path:     "/path/to/test/projects",
		IsActive: true,
	}

	err := AddRootFolder(rootFolder)
	if err != nil {
		t.Fatalf("AddRootFolder failed: %v", err)
	}

	if rootFolder.ID == 0 {
		t.Error("Root folder ID should be set after creation")
	}

	// Test 2: Get root folder by ID
	retrieved, err := GetRootFolderByID(rootFolder.ID)
	if err != nil {
		t.Fatalf("GetRootFolderByID failed: %v", err)
	}

	if retrieved.Name != rootFolder.Name {
		t.Errorf("Expected name %s, got %s", rootFolder.Name, retrieved.Name)
	}

	if retrieved.Path != rootFolder.Path {
		t.Errorf("Expected path %s, got %s", rootFolder.Path, retrieved.Path)
	}

	// Test 3: Get root folder by path
	byPath, err := GetRootFolderByPath(rootFolder.Path)
	if err != nil {
		t.Fatalf("GetRootFolderByPath failed: %v", err)
	}

	if byPath.ID != rootFolder.ID {
		t.Errorf("Expected ID %d, got %d", rootFolder.ID, byPath.ID)
	}

	// Test 4: Get active root folder
	active, err := GetActiveRootFolder()
	if err != nil {
		t.Fatalf("GetActiveRootFolder failed: %v", err)
	}

	if active.ID != rootFolder.ID {
		t.Errorf("Expected active folder ID %d, got %d", rootFolder.ID, active.ID)
	}

	// Test 5: Update root folder
	rootFolder.Name = "Updated Test Projects"
	rootFolder.GistID = "test-gist-id-123"
	err = UpdateRootFolder(rootFolder)
	if err != nil {
		t.Fatalf("UpdateRootFolder failed: %v", err)
	}

	updated, err := GetRootFolderByID(rootFolder.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated root folder: %v", err)
	}

	if updated.Name != "Updated Test Projects" {
		t.Errorf("Expected updated name, got %s", updated.Name)
	}

	if updated.GistID != "test-gist-id-123" {
		t.Errorf("Expected GistID test-gist-id-123, got %s", updated.GistID)
	}

	// Test 6: Add another root folder
	rootFolder2 := &models.RootFolder{
		Name:     "Another Projects",
		Path:     "/path/to/another/projects",
		IsActive: false,
	}

	err = AddRootFolder(rootFolder2)
	if err != nil {
		t.Fatalf("AddRootFolder (second) failed: %v", err)
	}

	// Test 7: Get all root folders
	allFolders, err := GetAllRootFolders()
	if err != nil {
		t.Fatalf("GetAllRootFolders failed: %v", err)
	}

	if len(allFolders) != 2 {
		t.Errorf("Expected 2 root folders, got %d", len(allFolders))
	}

	// Test 8: Set active root folder
	err = SetActiveRootFolder(rootFolder2.ID)
	if err != nil {
		t.Fatalf("SetActiveRootFolder failed: %v", err)
	}

	active, err = GetActiveRootFolder()
	if err != nil {
		t.Fatalf("GetActiveRootFolder after switch failed: %v", err)
	}

	if active.ID != rootFolder2.ID {
		t.Errorf("Expected active folder ID %d, got %d", rootFolder2.ID, active.ID)
	}

	// Verify first folder is no longer active
	first, err := GetRootFolderByID(rootFolder.ID)
	if err != nil {
		t.Fatalf("Failed to get first root folder: %v", err)
	}

	if first.IsActive {
		t.Error("First root folder should not be active after switching")
	}

	// Test 9: Delete root folder
	err = DeleteRootFolder(rootFolder2.ID)
	if err != nil {
		t.Fatalf("DeleteRootFolder failed: %v", err)
	}

	allFolders, err = GetAllRootFolders()
	if err != nil {
		t.Fatalf("GetAllRootFolders after delete failed: %v", err)
	}

	if len(allFolders) != 1 {
		t.Errorf("Expected 1 root folder after delete, got %d", len(allFolders))
	}
}

// TestProjectWithRootFolder tests project operations with root folders
func TestProjectWithRootFolder(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// Create a root folder
	rootFolder := &models.RootFolder{
		Name:     "Test Projects",
		Path:     "/path/to/projects",
		IsActive: true,
	}

	err := AddRootFolder(rootFolder)
	if err != nil {
		t.Fatalf("AddRootFolder failed: %v", err)
	}

	// Add projects to the root folder
	project1 := &models.Project{
		Name:         "Project 1",
		Path:         "/path/to/projects/project1",
		Status:       "active",
		LastOpened:   time.Now(),
		RootFolderID: rootFolder.ID,
	}

	err = AddProject(project1)
	if err != nil {
		t.Fatalf("AddProject failed: %v", err)
	}

	project2 := &models.Project{
		Name:         "Project 2",
		Path:         "/path/to/projects/project2",
		Status:       "active",
		LastOpened:   time.Now(),
		RootFolderID: rootFolder.ID,
	}

	err = AddProject(project2)
	if err != nil {
		t.Fatalf("AddProject (second) failed: %v", err)
	}

	// Get projects by root folder
	projects, err := GetProjectsByRootFolder(rootFolder.ID)
	if err != nil {
		t.Fatalf("GetProjectsByRootFolder failed: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Test GetProjects (should filter by active root folder)
	allProjects, err := GetProjects()
	if err != nil {
		t.Fatalf("GetProjects failed: %v", err)
	}

	if len(allProjects) != 2 {
		t.Errorf("Expected 2 projects from active root folder, got %d", len(allProjects))
	}

	// Create another root folder
	rootFolder2 := &models.RootFolder{
		Name:     "Other Projects",
		Path:     "/path/to/other",
		IsActive: false,
	}

	err = AddRootFolder(rootFolder2)
	if err != nil {
		t.Fatalf("AddRootFolder (second) failed: %v", err)
	}

	// Add a project to the second root folder
	project3 := &models.Project{
		Name:         "Project 3",
		Path:         "/path/to/other/project3",
		Status:       "active",
		LastOpened:   time.Now(),
		RootFolderID: rootFolder2.ID,
	}

	err = AddProject(project3)
	if err != nil {
		t.Fatalf("AddProject (third) failed: %v", err)
	}

	// GetProjects should still return 2 (from active root folder)
	allProjects, err = GetProjects()
	if err != nil {
		t.Fatalf("GetProjects after adding to second folder failed: %v", err)
	}

	if len(allProjects) != 2 {
		t.Errorf("Expected 2 projects from active root folder, got %d", len(allProjects))
	}

	// Switch active root folder
	err = SetActiveRootFolder(rootFolder2.ID)
	if err != nil {
		t.Fatalf("SetActiveRootFolder failed: %v", err)
	}

	// GetProjects should now return 1 (from new active root folder)
	allProjects, err = GetProjects()
	if err != nil {
		t.Fatalf("GetProjects after switch failed: %v", err)
	}

	if len(allProjects) != 1 {
		t.Errorf("Expected 1 project from new active root folder, got %d", len(allProjects))
	}

	if allProjects[0].Name != "Project 3" {
		t.Errorf("Expected Project 3, got %s", allProjects[0].Name)
	}

	// Test cascade delete: deleting a root folder should delete its projects
	err = DeleteRootFolder(rootFolder.ID)
	if err != nil {
		t.Fatalf("DeleteRootFolder failed: %v", err)
	}

	// Switch back to see the deleted projects
	err = SetActiveRootFolder(rootFolder2.ID)
	if err != nil {
		t.Fatalf("SetActiveRootFolder failed: %v", err)
	}

	// Try to get projects from deleted root folder
	deletedProjects, err := GetProjectsByRootFolder(rootFolder.ID)
	if err != nil {
		t.Fatalf("GetProjectsByRootFolder after delete failed: %v", err)
	}

	if len(deletedProjects) != 0 {
		t.Errorf("Expected 0 projects after root folder deletion, got %d", len(deletedProjects))
	}
}

// TestMain runs before all tests
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}
