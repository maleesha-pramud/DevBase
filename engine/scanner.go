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
