package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"devbase/models"

	_ "modernc.org/sqlite" // Use pure Go SQLite driver (no CGO required)
)

var DB *gorm.DB

// InitDB initializes the SQLite database connection with optimal performance settings
func InitDB(dbPath string) error {
	var err error

	// Configure GORM with performance optimizations
	config := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Silent),
		PrepareStmt: true, // Cache prepared statements for better performance
	}

	// Open SQLite connection using modernc.org/sqlite (pure Go, no CGO)
	// Add DSN parameters for proper datetime handling
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_time_format=sqlite"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Create GORM DB instance using the existing connection
	DB, err = gorm.Open(sqlite.Dialector{Conn: sqlDB}, config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// CRITICAL PERFORMANCE SETTINGS for SQLite

	// Enable Write-Ahead Logging (WAL) mode for better concurrency
	if err := DB.Exec("PRAGMA journal_mode = WAL;").Error; err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set synchronous mode to NORMAL for better performance
	// NORMAL is safe in WAL mode and much faster than FULL
	if err := DB.Exec("PRAGMA synchronous = NORMAL;").Error; err != nil {
		return fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	// Set connection pool settings to avoid SQLite locking issues
	sqlDB.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate the schema
	if err := DB.AutoMigrate(&models.RootFolder{}, &models.Project{}, &models.Config{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized successfully with WAL mode and optimized settings")
	return nil
}

// GetProjects retrieves all projects sorted by LastOpened descending
// If a root folder is active, only returns projects from that root folder
func GetProjects() ([]models.Project, error) {
	var projects []models.Project

	// Try to get active root folder
	activeRoot, err := GetActiveRootFolder()
	if err == nil && activeRoot != nil {
		// Filter by active root folder
		result := DB.Where("root_folder_id = ?", activeRoot.ID).Order("last_opened DESC").Find(&projects)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to retrieve projects: %w", result.Error)
		}
	} else {
		// No active root folder, return all projects
		result := DB.Order("last_opened DESC").Find(&projects)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to retrieve projects: %w", result.Error)
		}
	}

	return projects, nil
}

// AddProject adds a new project to the database
func AddProject(project *models.Project) error {
	// Set LastOpened to current time if not set
	if project.LastOpened.IsZero() {
		project.LastOpened = time.Now()
	}

	// Set default status if not provided
	if project.Status == "" {
		project.Status = "active"
	}

	// Validate status
	if project.Status != "active" && project.Status != "archived" {
		return fmt.Errorf("invalid status: must be 'active' or 'archived'")
	}

	result := DB.Create(project)
	if result.Error != nil {
		return fmt.Errorf("failed to add project: %w", result.Error)
	}
	return nil
}

// GetProjectByID retrieves a project by its ID
func GetProjectByID(id uint) (*models.Project, error) {
	var project models.Project
	result := DB.First(&project, id)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve project: %w", result.Error)
	}
	return &project, nil
}

// GetProjectByPath retrieves a project by its path
func GetProjectByPath(path string) (*models.Project, error) {
	var project models.Project
	result := DB.Where("path = ?", path).First(&project)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve project: %w", result.Error)
	}
	return &project, nil
}

// UpdateProject updates an existing project
func UpdateProject(project *models.Project) error {
	result := DB.Save(project)
	if result.Error != nil {
		return fmt.Errorf("failed to update project: %w", result.Error)
	}
	return nil
}

// DeleteProject soft deletes a project
func DeleteProject(id uint) error {
	result := DB.Delete(&models.Project{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete project: %w", result.Error)
	}
	return nil
}

// UpdateLastOpened updates the LastOpened timestamp for a project
func UpdateLastOpened(id uint) error {
	result := DB.Model(&models.Project{}).Where("id = ?", id).Update("last_opened", time.Now())
	if result.Error != nil {
		return fmt.Errorf("failed to update last_opened: %w", result.Error)
	}
	return nil
}

// DeleteAllProjects permanently deletes all projects and root folders from the database
func DeleteAllProjects() (int, error) {
	var count int64

	// Count projects before deletion
	if err := DB.Model(&models.Project{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Delete all projects (includes soft-deleted records)
	result := DB.Unscoped().Where("1 = 1").Delete(&models.Project{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete all projects: %w", result.Error)
	}

	// Delete all root folders as well
	if err := DB.Unscoped().Where("1 = 1").Delete(&models.RootFolder{}).Error; err != nil {
		return 0, fmt.Errorf("failed to delete all root folders: %w", err)
	}

	return int(count), nil
}

// CloseDB closes the database connection
func CloseDB() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

// GetConfig retrieves a configuration value by key
func GetConfig(key string) (string, error) {
	var config models.Config
	result := DB.Where("key = ?", key).First(&config)
	if result.Error != nil {
		return "", result.Error
	}
	return config.Value, nil
}

// SetConfig sets a configuration value
func SetConfig(key, value string) error {
	var config models.Config
	result := DB.Where("key = ?", key).First(&config)

	if result.Error == nil {
		// Update existing
		config.Value = value
		return DB.Save(&config).Error
	}

	// Create new
	config = models.Config{Key: key, Value: value}
	return DB.Create(&config).Error
}

// ========== RootFolder Management Functions ==========

// GetAllRootFolders retrieves all root folders
func GetAllRootFolders() ([]models.RootFolder, error) {
	var rootFolders []models.RootFolder
	result := DB.Order("created_at ASC").Find(&rootFolders)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve root folders: %w", result.Error)
	}
	return rootFolders, nil
}

// GetActiveRootFolder retrieves the currently active root folder
func GetActiveRootFolder() (*models.RootFolder, error) {
	var rootFolder models.RootFolder
	result := DB.Where("is_active = ?", true).First(&rootFolder)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve active root folder: %w", result.Error)
	}
	return &rootFolder, nil
}

// GetRootFolderByID retrieves a root folder by its ID
func GetRootFolderByID(id uint) (*models.RootFolder, error) {
	var rootFolder models.RootFolder
	result := DB.First(&rootFolder, id)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve root folder: %w", result.Error)
	}
	return &rootFolder, nil
}

// GetRootFolderByPath retrieves a root folder by its path
func GetRootFolderByPath(path string) (*models.RootFolder, error) {
	var rootFolder models.RootFolder
	result := DB.Where("path = ?", path).First(&rootFolder)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve root folder: %w", result.Error)
	}
	return &rootFolder, nil
}

// AddRootFolder adds a new root folder to the database
func AddRootFolder(rootFolder *models.RootFolder) error {
	result := DB.Create(rootFolder)
	if result.Error != nil {
		return fmt.Errorf("failed to add root folder: %w", result.Error)
	}
	return nil
}

// UpdateRootFolder updates an existing root folder
func UpdateRootFolder(rootFolder *models.RootFolder) error {
	result := DB.Save(rootFolder)
	if result.Error != nil {
		return fmt.Errorf("failed to update root folder: %w", result.Error)
	}
	return nil
}

// SetActiveRootFolder sets a root folder as active and deactivates all others
func SetActiveRootFolder(id uint) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Deactivate all root folders
		if err := tx.Model(&models.RootFolder{}).Where("1 = 1").Update("is_active", false).Error; err != nil {
			return fmt.Errorf("failed to deactivate root folders: %w", err)
		}

		// Activate the specified root folder
		if err := tx.Model(&models.RootFolder{}).Where("id = ?", id).Update("is_active", true).Error; err != nil {
			return fmt.Errorf("failed to activate root folder: %w", err)
		}

		return nil
	})
}

// DeleteRootFolder deletes a root folder and all its associated projects
func DeleteRootFolder(id uint) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Delete all projects in this root folder (hard delete to allow re-adding)
		if err := tx.Unscoped().Where("root_folder_id = ?", id).Delete(&models.Project{}).Error; err != nil {
			return fmt.Errorf("failed to delete projects: %w", err)
		}

		// Delete the root folder (hard delete to allow re-adding same path)
		if err := tx.Unscoped().Delete(&models.RootFolder{}, id).Error; err != nil {
			return fmt.Errorf("failed to delete root folder: %w", err)
		}

		return nil
	})
}

// GetProjectsByRootFolder retrieves all projects for a specific root folder
func GetProjectsByRootFolder(rootFolderID uint) ([]models.Project, error) {
	var projects []models.Project
	result := DB.Where("root_folder_id = ?", rootFolderID).Order("last_opened DESC").Find(&projects)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve projects: %w", result.Error)
	}
	return projects, nil
}
