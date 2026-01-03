package models

import (
	"time"

	"gorm.io/gorm"
)

// Config represents application configuration
type Config struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"unique;not null" json:"key"`
	Value string `json:"value"`
}

// RootFolder represents a root directory path where projects are stored
type RootFolder struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null" json:"name"`                    // User-friendly name for this root folder
	Path      string         `gorm:"not null;unique" json:"path"`             // Absolute path to the root folder
	IsActive  bool           `gorm:"not null;default:false" json:"is_active"` // Currently active root folder
	GistID    string         `json:"gist_id"`                                 // Gist ID for this root folder's cloud backup
	CreatedAt time.Time      `gorm:"type:datetime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:datetime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Projects  []Project      `gorm:"foreignKey:RootFolderID" json:"projects,omitempty"` // Projects in this root folder
}

// Project represents a development project in the database
type Project struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	Path         string         `gorm:"not null;uniqueIndex:idx_root_path" json:"path"` // Composite unique with RootFolderID
	RepoURL      string         `json:"repo_url"`
	Status       string         `gorm:"not null;default:active" json:"status"` // "active" or "archived"
	LastOpened   time.Time      `gorm:"not null;type:datetime" json:"last_opened"`
	Tags         []string       `gorm:"serializer:json" json:"tags"`
	RootFolderID uint           `gorm:"default:0;index;uniqueIndex:idx_root_path" json:"root_folder_id"` // Foreign key to RootFolder, composite unique with Path
	CreatedAt    time.Time      `gorm:"type:datetime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"type:datetime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
