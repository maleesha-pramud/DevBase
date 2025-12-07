package engine

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"devbase/models"
)

// ScanDirectory concurrently scans a root directory for projects and returns discovered projects.
// A worker pool evaluates directories for project markers (package.json, go.mod, .git).
func ScanDirectory(rootPath string) ([]models.Project, error) {
	const workerCount = 10
	jobs := make(chan string, workerCount*4)
	results := make(chan models.Project, workerCount*4)

	// Worker pool to process directory paths.
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range jobs {
				if project, ok, err := inspectDirectory(dir); err == nil && ok {
					results <- project
				}
			}
		}()
	}

	// Walk the directory tree on the main goroutine and dispatch work.
	ignore := map[string]struct{}{
		"node_modules": {},
		"dist":         {},
		"build":        {},
		"vendor":       {},
		".next":        {},
		".vite":        {},
	}

	walkErr := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if _, skip := ignore[name]; skip {
			return filepath.SkipDir // prune heavy directories early
		}

		jobs <- path
		return nil
	})

	close(jobs)
	wg.Wait()
	close(results)

	if walkErr != nil {
		return nil, walkErr
	}

	// Collect results
	var projects []models.Project
	seen := make(map[string]struct{})
	for p := range results {
		if _, exists := seen[p.Path]; exists {
			continue
		}
		seen[p.Path] = struct{}{}
		projects = append(projects, p)
	}

	return projects, nil
}

// inspectDirectory checks if a directory contains project markers and constructs a Project.
func inspectDirectory(dir string) (models.Project, bool, error) {
	markers := []string{"package.json", "go.mod", ".git"}
	for _, m := range markers {
		if exists, err := fileExists(filepath.Join(dir, m)); err != nil {
			return models.Project{}, false, err
		} else if exists {
			project := models.Project{
				Name:       filepath.Base(dir),
				Path:       dir,
				Status:     "active",
				LastOpened: time.Now(),
			}

			// Try to get git remote URL
			if gitURL := getGitRemoteURL(dir); gitURL != "" {
				project.RepoURL = gitURL
			}

			return project, true, nil
		}
	}
	return models.Project{}, false, nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		_ = info
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// getGitRemoteURL extracts the git remote origin URL from a directory
func getGitRemoteURL(dir string) string {
	gitConfigPath := filepath.Join(dir, ".git", "config")

	// Check if .git/config exists
	if exists, _ := fileExists(gitConfigPath); !exists {
		return ""
	}

	// Read git config file
	data, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return ""
	}

	// Parse the config to find origin URL
	lines := string(data)
	var inOriginSection bool

	for _, line := range splitLines(lines) {
		trimmed := trimSpace(line)

		// Check if we're entering the [remote "origin"] section
		if trimmed == `[remote "origin"]` {
			inOriginSection = true
			continue
		}

		// Check if we're leaving the origin section
		if inOriginSection && len(trimmed) > 0 && trimmed[0] == '[' {
			inOriginSection = false
			continue
		}

		// Look for url = line in origin section
		if inOriginSection && startsWithIgnoreSpace(trimmed, "url") {
			// Extract URL after "url = "
			if idx := indexByte(trimmed, '='); idx >= 0 && idx+1 < len(trimmed) {
				url := trimSpace(trimmed[idx+1:])
				return url
			}
		}
	}

	return ""
}

// Helper functions to avoid importing strings package
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}

	return s[start:end]
}

func startsWithIgnoreSpace(s, prefix string) bool {
	s = trimSpace(s)
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
