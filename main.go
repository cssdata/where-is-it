package main

import (
	"log"
	"net/http"

	"github.com/cssdata/where-is-it/internal/handlers"
	"github.com/cssdata/where-is-it/internal/storage"
)

func main() {
	// Initialize storage
	store, err := storage.NewJSONStorage("./data")
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Initialize handlers
	handler := handlers.NewHandler(store)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Web routes
	mux.HandleFunc("/", handler.Home)
	mux.HandleFunc("/locations", handler.Locations)
	mux.HandleFunc("/locations/new", handler.CreateLocation)
	mux.HandleFunc("/items", handler.Items)
	mux.HandleFunc("/items/new", handler.CreateItem)
	mux.HandleFunc("/search", handler.Search)

	// API routes
	mux.HandleFunc("/api/locations", handler.APILocations)
	mux.HandleFunc("/api/items", handler.APIItems)
	mux.HandleFunc("/api/search", handler.APISearch)
	mux.HandleFunc("/api/positions", handler.APIPositions)

	log.Println("Starting server on 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", mux))
}
