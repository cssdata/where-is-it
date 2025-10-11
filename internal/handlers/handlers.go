package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/cssdata/where-is-it/internal/models"
	"github.com/cssdata/where-is-it/internal/storage"
)

type Handler struct {
	storage storage.Storage
	tmpl    *template.Template
}

func NewHandler(store storage.Storage) *Handler {
	// Parse templates
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Handler{
		storage: store,
		tmpl:    tmpl,
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
		h.tmpl.ExecuteTemplate(w, "locations.html", locations)
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
		locationID := r.URL.Query().Get("location")
		var items []models.Item
		var err error

		if locationID != "" {
			items, err = h.storage.GetItemsByLocation(locationID)
		} else {
			items, err = h.storage.GetAllItems()
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get locations for display
		locations, _ := h.storage.GetAllLocations()
		locationMap := make(map[string]models.Location)
		for _, loc := range locations {
			locationMap[loc.ID] = loc
		}

		data := struct {
			Items     []models.Item
			Locations map[string]models.Location
		}{
			Items:     items,
			Locations: locationMap,
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

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		locations, _ := h.storage.GetAllLocations()
		h.tmpl.ExecuteTemplate(w, "item-form.html", locations)
	case "POST":
		quantity, _ := strconv.Atoi(r.FormValue("quantity"))

		item := &models.Item{
			Name:        r.FormValue("name"),
			Description: r.FormValue("description"),
			LocationID:  r.FormValue("location_id"),
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

		http.Redirect(w, r, "/items", http.StatusSeeOther)
	}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var results []models.SearchResult

	if query != "" {
		searchQuery := models.SearchQuery{
			Query: query,
		}
		var err error
		results, err = h.storage.SearchItems(searchQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		Query   string
		Results []models.SearchResult
	}{
		Query:   query,
		Results: results,
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
