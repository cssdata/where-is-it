package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/cssdata/where-is-it/internal/models"
	"github.com/cssdata/where-is-it/internal/storage"
)

type Handler struct {
	storage storage.Storage
	tmpl    *template.Template
	// Default values for new items
	lastUsedDefaults *ItemDefaults
}

type ItemDefaults struct {
	LocationID       string
	Position         string
	Unit             string
	Properties       map[string]string
	FilterLocationID string // For filtering items by location
}

func NewHandler(store storage.Storage) *Handler {
	// Parse templates
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Handler{
		storage:          store,
		tmpl:             tmpl,
		lastUsedDefaults: &ItemDefaults{Unit: "Stück", Properties: make(map[string]string)},
	}
}

// Web handlers
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	locations, _ := h.storage.GetAllLocations()
	items, _ := h.storage.GetAllItems()

	data := struct {
		LocationCount int
		ItemCount     int
	}{
		LocationCount: len(locations),
		ItemCount:     len(items),
	}

	h.tmpl.ExecuteTemplate(w, "home.html", data)
}

func (h *Handler) Locations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		locations, err := h.storage.GetAllLocations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get all items to check which locations have items
		allItems, err := h.storage.GetAllItems()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create maps for checking dependencies
		locationHasItems := make(map[string]bool)
		locationHasChildren := make(map[string]bool)

		// Check which locations have items
		for _, item := range allItems {
			locationHasItems[item.LocationID] = true
		}

		// Check which locations have children
		for _, location := range locations {
			if location.ParentID != nil {
				locationHasChildren[*location.ParentID] = true
			}
		}

		data := struct {
			Locations           []models.Location
			LocationHasItems    map[string]bool
			LocationHasChildren map[string]bool
		}{
			Locations:           locations,
			LocationHasItems:    locationHasItems,
			LocationHasChildren: locationHasChildren,
		}

		h.tmpl.ExecuteTemplate(w, "locations.html", data)
	case "DELETE":
		// Handle location deletion
		locationID := r.URL.Query().Get("id")
		if locationID == "" {
			http.Error(w, "Location ID required", http.StatusBadRequest)
			return
		}

		// Check if location has items
		items, err := h.storage.GetItemsByLocation(locationID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(items) > 0 {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("Cannot delete location: it contains items"))
			return
		}

		// Check if location has child locations
		allLocations, err := h.storage.GetAllLocations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		hasChildren := false
		for _, loc := range allLocations {
			if loc.ParentID != nil && *loc.ParentID == locationID {
				hasChildren = true
				break
			}
		}

		if hasChildren {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("Cannot delete location: it has sub-locations"))
			return
		}

		if err := h.storage.DeleteLocation(locationID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Location deleted successfully"))
	}
}

func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		locations, _ := h.storage.GetAllLocations()
		h.tmpl.ExecuteTemplate(w, "location-form.html", locations)
	case "POST":
		location := &models.Location{
			Name:        r.FormValue("name"),
			Description: r.FormValue("description"),
		}

		if parentID := r.FormValue("parent_id"); parentID != "" {
			location.ParentID = &parentID
		}

		if err := h.storage.CreateLocation(location); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/locations", http.StatusSeeOther)
	}
}

func (h *Handler) Items(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		locationFilter := r.URL.Query().Get("location")
		
		// If no location filter is specified, use the default filter location
		if locationFilter == "" {
			locationFilter = h.lastUsedDefaults.FilterLocationID
		}
		
		var items []models.Item
		var err error

		// Get all items first
		items, err = h.storage.GetAllItems()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get all locations for hierarchical filtering
		locations, err := h.storage.GetAllLocations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create location map and find child locations for hierarchical filtering
		locationMap := make(map[string]models.Location)
		for _, loc := range locations {
			locationMap[loc.ID] = loc
		}

		// Filter by location if specified (including child locations)
		if locationFilter != "" {
			var filteredItems []models.Item
			childLocationIDs := h.getChildLocationIDs(locationFilter, locations)
			childLocationIDs[locationFilter] = true // Include the filter location itself
			
			for _, item := range items {
				if childLocationIDs[item.LocationID] {
					filteredItems = append(filteredItems, item)
				}
			}
			items = filteredItems
		}

		// Prepare locations for dropdown (only top-level and their immediate children)
		var dropdownLocations []models.Location
		for _, loc := range locations {
			// Include all locations for the dropdown
			dropdownLocations = append(dropdownLocations, loc)
		}
		sort.Slice(dropdownLocations, func(i, j int) bool {
			return dropdownLocations[i].Path < dropdownLocations[j].Path
		})

		data := struct {
			Items            []models.Item
			Locations        map[string]models.Location
			DropdownLocations []models.Location
			SelectedLocation string
		}{
			Items:             items,
			Locations:         locationMap,
			DropdownLocations: dropdownLocations,
			SelectedLocation:  locationFilter,
		}

		h.tmpl.ExecuteTemplate(w, "items.html", data)
	case "DELETE":
		// Handle item deletion
		itemID := r.URL.Query().Get("id")
		if itemID == "" {
			http.Error(w, "Item ID required", http.StatusBadRequest)
			return
		}

		if err := h.storage.DeleteItem(itemID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Item deleted successfully"))
	}
}

// Helper function to get all child location IDs for hierarchical filtering
func (h *Handler) getChildLocationIDs(parentID string, allLocations []models.Location) map[string]bool {
	childIDs := make(map[string]bool)
	
	// Find direct children
	for _, loc := range allLocations {
		if loc.ParentID != nil && *loc.ParentID == parentID {
			childIDs[loc.ID] = true
			// Recursively find children of children
			grandChildIDs := h.getChildLocationIDs(loc.ID, allLocations)
			for grandChildID := range grandChildIDs {
				childIDs[grandChildID] = true
			}
		}
	}
	
	return childIDs
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		locations, _ := h.storage.GetAllLocations()

		// If a location is pre-selected (e.g., from location page), get its positions
		selectedLocationID := r.URL.Query().Get("location")
		if selectedLocationID == "" {
			// Use last used location as default if no specific location is requested
			selectedLocationID = h.lastUsedDefaults.LocationID
		}

		var positions []string
		if selectedLocationID != "" {
			positions, _ = h.storage.GetPositionsByLocation(selectedLocationID)
		}

		data := struct {
			Locations          []models.Location
			SelectedLocationID string
			Positions          []string
			DefaultPosition    string
			DefaultUnit        string
			DefaultProperties  map[string]string
		}{
			Locations:          locations,
			SelectedLocationID: selectedLocationID,
			Positions:          positions,
			DefaultPosition:    h.lastUsedDefaults.Position,
			DefaultUnit:        h.lastUsedDefaults.Unit,
			DefaultProperties:  h.lastUsedDefaults.Properties,
		}

		h.tmpl.ExecuteTemplate(w, "item-form.html", data)
	case "POST":
		quantity, _ := strconv.Atoi(r.FormValue("quantity"))

		item := &models.Item{
			Name:        r.FormValue("name"),
			Description: r.FormValue("description"),
			LocationID:  r.FormValue("location_id"),
			Position:    r.FormValue("position"),
			Quantity:    quantity,
			Unit:        r.FormValue("unit"),
			Properties:  make(map[string]string),
			Tags:        []string{},
		}

		// Parse properties from JSON
		if props := r.FormValue("properties"); props != "" {
			json.Unmarshal([]byte(props), &item.Properties)
		}

		// Parse tags (comma-separated)
		if tags := r.FormValue("tags"); tags != "" {
			for _, tag := range strings.Split(tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					item.Tags = append(item.Tags, tag)
				}
			}
		}

		if err := h.storage.CreateItem(item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update the default values for next time
		h.lastUsedDefaults.LocationID = item.LocationID
		h.lastUsedDefaults.Position = item.Position
		h.lastUsedDefaults.Unit = item.Unit
		// Set the filter location to the newly created item's location
		h.lastUsedDefaults.FilterLocationID = item.LocationID
		// Make a copy of properties to avoid reference issues
		h.lastUsedDefaults.Properties = make(map[string]string)
		for k, v := range item.Properties {
			h.lastUsedDefaults.Properties[k] = v
		}

		http.Redirect(w, r, "/items", http.StatusSeeOther)
	}
}

func (h *Handler) EditItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		itemID := r.URL.Query().Get("id")
		if itemID == "" {
			http.Error(w, "Item ID required", http.StatusBadRequest)
			return
		}

		item, err := h.storage.GetItem(itemID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		locations, _ := h.storage.GetAllLocations()
		positions, _ := h.storage.GetPositionsByLocation(item.LocationID)

		data := struct {
			Item      models.Item
			Locations []models.Location
			Positions []string
		}{
			Item:      *item,
			Locations: locations,
			Positions: positions,
		}

		h.tmpl.ExecuteTemplate(w, "item-edit-form.html", data)
	case "POST":
		itemID := r.URL.Query().Get("id")
		if itemID == "" {
			http.Error(w, "Item ID required", http.StatusBadRequest)
			return
		}

		quantity, _ := strconv.Atoi(r.FormValue("quantity"))

		item := &models.Item{
			ID:          itemID,
			Name:        r.FormValue("name"),
			Description: r.FormValue("description"),
			LocationID:  r.FormValue("location_id"),
			Position:    r.FormValue("position"),
			Quantity:    quantity,
			Unit:        r.FormValue("unit"),
			Properties:  make(map[string]string),
			Tags:        []string{},
		}

		// Parse properties from JSON
		if props := r.FormValue("properties"); props != "" {
			json.Unmarshal([]byte(props), &item.Properties)
		}

		// Parse tags (comma-separated)
		if tags := r.FormValue("tags"); tags != "" {
			for _, tag := range strings.Split(tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					item.Tags = append(item.Tags, tag)
				}
			}
		}

		if err := h.storage.UpdateItem(item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/items", http.StatusSeeOther)
	}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	locationFilter := r.URL.Query().Get("location")
	
	var results []models.SearchResult

	if query != "" {
		searchQuery := models.SearchQuery{
			Query: query,
			// Don't pass LocationID to storage, we'll filter hierarchically ourselves
		}
		var err error
		results, err = h.storage.SearchItems(searchQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Apply hierarchical location filtering if specified
		if locationFilter != "" {
			locations, err := h.storage.GetAllLocations()
			if err == nil {
				childLocationIDs := h.getChildLocationIDs(locationFilter, locations)
				childLocationIDs[locationFilter] = true // Include the filter location itself
				
				var filteredResults []models.SearchResult
				for _, result := range results {
					if childLocationIDs[result.Item.LocationID] {
						filteredResults = append(filteredResults, result)
					}
				}
				results = filteredResults
			}
		}
	}

	// Get all locations for the dropdown
	locations, _ := h.storage.GetAllLocations()
	var dropdownLocations []models.Location
	for _, loc := range locations {
		dropdownLocations = append(dropdownLocations, loc)
	}
	sort.Slice(dropdownLocations, func(i, j int) bool {
		return dropdownLocations[i].Path < dropdownLocations[j].Path
	})

	data := struct {
		Query             string
		Results           []models.SearchResult
		DropdownLocations []models.Location
		SelectedLocation  string
	}{
		Query:             query,
		Results:           results,
		DropdownLocations: dropdownLocations,
		SelectedLocation:  locationFilter,
	}

	h.tmpl.ExecuteTemplate(w, "search.html", data)
}

// API handlers
func (h *Handler) APILocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		locations, err := h.storage.GetAllLocations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(locations)
	case "POST":
		var location models.Location
		if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.storage.CreateLocation(&location); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(location)
	}
}

func (h *Handler) APIItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		items, err := h.storage.GetAllItems()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(items)
	case "POST":
		var item models.Item
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.storage.CreateItem(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	}
}

func (h *Handler) APIPositions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	locationID := r.URL.Query().Get("location_id")
	if locationID == "" {
		http.Error(w, "location_id parameter required", http.StatusBadRequest)
		return
	}

	positions, err := h.storage.GetPositionsByLocation(locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(positions)
}

func (h *Handler) APISearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var searchQuery models.SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&searchQuery); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := h.storage.SearchItems(searchQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Handler) APIPropertyNames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	items, err := h.storage.GetAllItems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Collect all unique property names
	propertyNames := make(map[string]bool)
	for _, item := range items {
		for key := range item.Properties {
			propertyNames[key] = true
		}
	}

	// Convert to sorted slice
	var names []string
	for name := range propertyNames {
		names = append(names, name)
	}
	sort.Strings(names)

	json.NewEncoder(w).Encode(names)
}
