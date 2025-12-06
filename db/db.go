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
	if err := DB.AutoMigrate(&models.Project{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized successfully with WAL mode and optimized settings")
	return nil
}

// GetProjects retrieves all projects sorted by LastOpened descending
func GetProjects() ([]models.Project, error) {
	var projects []models.Project
	result := DB.Order("last_opened DESC").Find(&projects)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve projects: %w", result.Error)
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

// DeleteAllProjects permanently deletes all projects from the database
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
