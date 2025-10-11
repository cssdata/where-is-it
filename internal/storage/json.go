package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cssdata/where-is-it/internal/models"
	"github.com/google/uuid"
)

type Storage interface {
	// Location methods
	CreateLocation(location *models.Location) error
	GetLocation(id string) (*models.Location, error)
	GetAllLocations() ([]models.Location, error)
	UpdateLocation(location *models.Location) error
	DeleteLocation(id string) error

	// Item methods
	CreateItem(item *models.Item) error
	GetItem(id string) (*models.Item, error)
	GetAllItems() ([]models.Item, error)
	GetItemsByLocation(locationID string) ([]models.Item, error)
	GetPositionsByLocation(locationID string) ([]string, error)
	UpdateItem(item *models.Item) error
	DeleteItem(id string) error

	// Search methods
	SearchItems(query models.SearchQuery) ([]models.SearchResult, error)
}

type JSONStorage struct {
	dataDir   string
	locations map[string]models.Location
	items     map[string]models.Item
}

func NewJSONStorage(dataDir string) (*JSONStorage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	storage := &JSONStorage{
		dataDir:   dataDir,
		locations: make(map[string]models.Location),
		items:     make(map[string]models.Item),
	}

	// Load existing data
	if err := storage.loadData(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *JSONStorage) loadData() error {
	// Load locations
	locationsFile := filepath.Join(s.dataDir, "locations.json")
	if data, err := os.ReadFile(locationsFile); err == nil {
		var locations []models.Location
		if err := json.Unmarshal(data, &locations); err == nil {
			for _, loc := range locations {
				s.locations[loc.ID] = loc
			}
		}
	}

	// Load items
	itemsFile := filepath.Join(s.dataDir, "items.json")
	if data, err := os.ReadFile(itemsFile); err == nil {
		var items []models.Item
		if err := json.Unmarshal(data, &items); err == nil {
			for _, item := range items {
				s.items[item.ID] = item
			}
		}
	}

	return nil
}

func (s *JSONStorage) saveData() error {
	// Save locations
	var locations []models.Location
	for _, loc := range s.locations {
		locations = append(locations, loc)
	}
	locationsData, err := json.MarshalIndent(locations, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, "locations.json"), locationsData, 0644); err != nil {
		return err
	}

	// Save items
	var items []models.Item
	for _, item := range s.items {
		items = append(items, item)
	}
	itemsData, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dataDir, "items.json"), itemsData, 0644)
}

// Location methods
func (s *JSONStorage) CreateLocation(location *models.Location) error {
	location.ID = uuid.New().String()
	location.CreatedAt = time.Now()
	location.UpdatedAt = time.Now()

	// Build path
	if location.ParentID != nil {
		if parent, exists := s.locations[*location.ParentID]; exists {
			location.Path = parent.Path + "/" + location.Name
		} else {
			return fmt.Errorf("parent location not found")
		}
	} else {
		location.Path = location.Name
	}

	s.locations[location.ID] = *location
	return s.saveData()
}

func (s *JSONStorage) GetLocation(id string) (*models.Location, error) {
	if location, exists := s.locations[id]; exists {
		return &location, nil
	}
	return nil, fmt.Errorf("location not found")
}

func (s *JSONStorage) GetAllLocations() ([]models.Location, error) {
	var locations []models.Location
	for _, loc := range s.locations {
		locations = append(locations, loc)
	}

	// Sort by path for hierarchical display
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Path < locations[j].Path
	})

	return locations, nil
}

func (s *JSONStorage) UpdateLocation(location *models.Location) error {
	if _, exists := s.locations[location.ID]; !exists {
		return fmt.Errorf("location not found")
	}
	location.UpdatedAt = time.Now()
	s.locations[location.ID] = *location
	return s.saveData()
}

func (s *JSONStorage) DeleteLocation(id string) error {
	// Check if any items reference this location
	for _, item := range s.items {
		if item.LocationID == id {
			return fmt.Errorf("cannot delete location: items still reference it")
		}
	}

	delete(s.locations, id)
	return s.saveData()
}

// Item methods
func (s *JSONStorage) CreateItem(item *models.Item) error {
	item.ID = uuid.New().String()
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	if item.Properties == nil {
		item.Properties = make(map[string]string)
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}

	s.items[item.ID] = *item
	return s.saveData()
}

func (s *JSONStorage) GetItem(id string) (*models.Item, error) {
	if item, exists := s.items[id]; exists {
		return &item, nil
	}
	return nil, fmt.Errorf("item not found")
}

func (s *JSONStorage) GetAllItems() ([]models.Item, error) {
	var items []models.Item
	for _, item := range s.items {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (s *JSONStorage) GetItemsByLocation(locationID string) ([]models.Item, error) {
	var items []models.Item
	for _, item := range s.items {
		if item.LocationID == locationID {
			items = append(items, item)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (s *JSONStorage) GetPositionsByLocation(locationID string) ([]string, error) {
	positionSet := make(map[string]bool)
	var positions []string

	for _, item := range s.items {
		if item.LocationID == locationID && item.Position != "" {
			if !positionSet[item.Position] {
				positionSet[item.Position] = true
				positions = append(positions, item.Position)
			}
		}
	}

	sort.Strings(positions)
	return positions, nil
}

func (s *JSONStorage) UpdateItem(item *models.Item) error {
	if _, exists := s.items[item.ID]; !exists {
		return fmt.Errorf("item not found")
	}
	item.UpdatedAt = time.Now()
	s.items[item.ID] = *item
	return s.saveData()
}

func (s *JSONStorage) DeleteItem(id string) error {
	delete(s.items, id)
	return s.saveData()
}

// Search methods
func (s *JSONStorage) SearchItems(query models.SearchQuery) ([]models.SearchResult, error) {
	var results []models.SearchResult

	for _, item := range s.items {
		if s.matchesQuery(item, query) {
			if location, exists := s.locations[item.LocationID]; exists {
				results = append(results, models.SearchResult{
					Item:     item,
					Location: location,
				})
			}
		}
	}

	return results, nil
}

func (s *JSONStorage) matchesQuery(item models.Item, query models.SearchQuery) bool {
	// Free text search
	if query.Query != "" {
		searchText := strings.ToLower(query.Query)
		if strings.Contains(strings.ToLower(item.Name), searchText) ||
			strings.Contains(strings.ToLower(item.Description), searchText) {
			// Match found in name or description
		} else {
			// Check properties
			found := false
			for key, value := range item.Properties {
				if strings.Contains(strings.ToLower(key), searchText) ||
					strings.Contains(strings.ToLower(value), searchText) {
					found = true
					break
				}
			}
			// Check tags
			if !found {
				for _, tag := range item.Tags {
					if strings.Contains(strings.ToLower(tag), searchText) {
						found = true
						break
					}
				}
			}
			if !found {
				return false
			}
		}
	}

	// Location filter
	if query.LocationID != "" && item.LocationID != query.LocationID {
		return false
	}

	// Tags filter
	if len(query.Tags) > 0 {
		hasTag := false
		for _, queryTag := range query.Tags {
			for _, itemTag := range item.Tags {
				if strings.EqualFold(queryTag, itemTag) {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	// Properties filter
	for key, value := range query.Properties {
		if itemValue, exists := item.Properties[key]; !exists || !strings.EqualFold(itemValue, value) {
			return false
		}
	}

	return true
}
