package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v5"

	"devbase/db"
)

// ArchiveProject archives a project by updating its status and deleting the physical directory
func ArchiveProject(projectID uint) error {
	// Retrieve the project from the database
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Verify the path exists before attempting deletion
	if _, err := os.Stat(project.Path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat project path: %w", err)
		}
		// Path doesn't exist, but we'll still update the status
	} else {
		// Path exists, delete it recursively
		if err := os.RemoveAll(project.Path); err != nil {
			return fmt.Errorf("failed to delete project directory at %s: %w", project.Path, err)
		}
	}

	// Update the project status to "archived" in the database
	project.Status = "archived"
	if err := db.UpdateProject(project); err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}

	return nil
}

// RestoreProject restores a project by cloning its repository and updating the status
func RestoreProject(projectID uint) error {
	// Retrieve the project from the database
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Validate that the project has a RepoURL
	if project.RepoURL == "" {
		return fmt.Errorf("project %s has no repository URL", project.Name)
	}

	// Ensure the directory does not currently exist
	if _, err := os.Stat(project.Path); err == nil {
		return fmt.Errorf("project directory already exists at %s", project.Path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check project path: %w", err)
	}

	// Create the parent directory if it doesn't exist
	parentDir := filepath.Dir(project.Path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// For private repositories, we need to use system git with credential helper
	// The go-git library doesn't easily integrate with Windows Credential Manager
	// So we'll fall back to using system git command for authentication

	// Try using system git command which has credential helper configured
	err = cloneWithSystemGit(project.RepoURL, project.Path)
	if err != nil {
		// Clean up the directory if clone fails
		_ = os.RemoveAll(project.Path)
		return fmt.Errorf("failed to clone repository from %s: %w", project.RepoURL, err)
	}

	// Update the project status to "active" in the database
	project.Status = "active"
	if err := db.UpdateProject(project); err != nil {
		// Attempt to clean up on update failure
		_ = os.RemoveAll(project.Path)
		return fmt.Errorf("failed to update project status: %w", err)
	}

	// Update the LastOpened timestamp
	if err := db.UpdateLastOpened(projectID); err != nil {
		return fmt.Errorf("failed to update last opened timestamp: %w", err)
	}

	return nil
}

// DeleteProjectPermanently completely removes a project (DB record + files)
// WARNING: This is destructive and cannot be undone
func DeleteProjectPermanently(projectID uint) error {
	// Retrieve the project from the database
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Delete the physical directory if it exists
	if _, err := os.Stat(project.Path); err == nil {
		if err := os.RemoveAll(project.Path); err != nil {
			return fmt.Errorf("failed to delete project directory: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check project path: %w", err)
	}

	// Delete the database record
	if err := db.DeleteProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project from database: %w", err)
	}

	return nil
}

// GetProjectStatus retrieves the current status of a project
func GetProjectStatus(projectID uint) (string, error) {
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve project: %w", err)
	}
	return project.Status, nil
}

// VerifyProjectPath checks if a project's directory exists
func VerifyProjectPath(projectID uint) (bool, error) {
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve project: %w", err)
	}

	_, err = os.Stat(project.Path)
	if err == nil {
		return true, nil // Path exists
	} else if os.IsNotExist(err) {
		return false, nil // Path does not exist
	}
	return false, err // Error checking path
}

// GetLatestCommitHash retrieves the latest commit hash of a project's git repository
func GetLatestCommitHash(projectID uint) (string, error) {
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Open the git repository
	repo, err := git.PlainOpen(project.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get the HEAD reference
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return head.Hash().String(), nil
}

// IsManagedByGit checks if a project directory is a valid git repository
func IsManagedByGit(projectID uint) (bool, error) {
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve project: %w", err)
	}

	_, err = git.PlainOpen(project.Path)
	if err == nil {
		return true, nil // Valid git repository
	} else if err == git.ErrRepositoryNotExists {
		return false, nil // Not a git repository
	}
	return false, err // Error checking repository
}

// ArchiveWithVerification archives a project with comprehensive verification
func ArchiveWithVerification(projectID uint) error {
	// Verify the project exists
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Skip if already archived
	if project.Status == "archived" {
		return fmt.Errorf("project is already archived")
	}

	// Verify the path exists and is readable
	if _, err := os.Stat(project.Path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot access project path: %w", err)
		}
	}

	// Proceed with archival
	return ArchiveProject(projectID)
}

// RestoreWithVerification restores a project with comprehensive verification
func RestoreWithVerification(projectID uint) error {
	// Verify the project exists
	project, err := db.GetProjectByID(projectID)
	if err != nil {
		return fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Skip if already active
	if project.Status == "active" {
		return fmt.Errorf("project is already active")
	}

	// Verify the project has a repository URL
	if project.RepoURL == "" {
		return fmt.Errorf("cannot restore project without a repository URL")
	}

	// Proceed with restoration
	return RestoreProject(projectID)
}

// cloneWithSystemGit uses the system's git command to clone a repository
// This allows using the system's credential helper (Windows Credential Manager, etc.)
func cloneWithSystemGit(repoURL, destPath string) error {
	// Use git clone with depth 1 for faster cloning
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, destPath)

	// Capture output for better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}

	return nil
}
