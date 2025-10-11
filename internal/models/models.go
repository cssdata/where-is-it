package models

import (
	"time"
)

// Location represents a hierarchical place where items can be stored
type Location struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *string   `json:"parent_id,omitempty"` // For hierarchical locations
	Path        string    `json:"path"`                // Full path like "Cellar/Werkstatt/Cupboard-2"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Item represents something stored at a location
type Item struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	LocationID  string            `json:"location_id"`
	Quantity    int               `json:"quantity"`
	Unit        string            `json:"unit"`       // e.g., "pieces", "kg", "liters"
	Properties  map[string]string `json:"properties"` // e.g., {"size": "M4", "length": "20mm", "standard": "ISO XYZ"}
	Tags        []string          `json:"tags"`       // e.g., ["screws", "fasteners", "hardware"]
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SearchResult combines item and location information for search results
type SearchResult struct {
	Item     Item     `json:"item"`
	Location Location `json:"location"`
}

// SearchQuery represents search parameters
type SearchQuery struct {
	Query      string            `json:"query"`       // Free text search
	LocationID string            `json:"location_id"` // Filter by location
	Tags       []string          `json:"tags"`        // Filter by tags
	Properties map[string]string `json:"properties"`  // Filter by properties
}
