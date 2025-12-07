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

// Project represents a development project in the database
type Project struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"not null" json:"name"`
	Path       string         `gorm:"not null;unique" json:"path"`
	RepoURL    string         `json:"repo_url"`
	Status     string         `gorm:"not null;default:active" json:"status"` // "active" or "archived"
	LastOpened time.Time      `gorm:"not null;type:datetime" json:"last_opened"`
	Tags       []string       `gorm:"serializer:json" json:"tags"`
	CreatedAt  time.Time      `gorm:"type:datetime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"type:datetime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
