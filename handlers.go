package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"streets": fmt.Sprintf("%d", len(service.streets)),
		"cities":  fmt.Sprintf("%d", len(service.cities)),
	})
}

func streetsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode(AutocompleteResponse{
			Query:   "",
			Results: []StreetRecord{},
			Count:   0,
			Time:    time.Since(startTime).String(),
		})
		return
	}

	// Default limit
	limit := 10

	// Search
	results := service.SearchStreets(query, limit)

	// Build response
	response := AutocompleteResponse{
		Query:   query,
		Results: results,
		Count:   len(results),
		Time:    time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}

func citiesHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get query parameter
	query := r.URL.Query().Get("q")

	// Get filter parameters (0 means no filter)
	woj, _ := strconv.Atoi(r.URL.Query().Get("woj"))
	pow, _ := strconv.Atoi(r.URL.Query().Get("pow"))
	gmi, _ := strconv.Atoi(r.URL.Query().Get("gmi"))

	// Default limit
	limit := 10

	// Search with filters
	results := service.SearchCities(query, woj, pow, gmi, limit)

	// Build filters map for response
	filters := make(map[string]int)
	if woj > 0 {
		filters["woj"] = woj
	}
	if pow > 0 {
		filters["pow"] = pow
	}
	if gmi > 0 {
		filters["gmi"] = gmi
	}

	// Build response
	response := CityAutocompleteResponse{
		Query:   query,
		Filters: filters,
		Results: results,
		Count:   len(results),
		Time:    time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}

// streetGMIHandler handles HTTP requests to retrieve GMI codes for a specific street name.
// It expects a 'name' query parameter containing the street name.
// Returns a JSON response with the street name, matching GMI codes, result count, and processing time.
// If the 'name' parameter is missing, returns an error response.
func streetGMIHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get street name parameter
	streetName := r.URL.Query().Get("name")
	if streetName == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "missing 'name' parameter",
			"results": []map[string]interface{}{},
			"count":   0,
			"time":    time.Since(startTime).String(),
		})
		return
	}

	// Get GMI codes for the exact street name
	results := service.GetGMIForStreet(streetName)

	// Build response
	response := map[string]interface{}{
		"street_name": streetName,
		"results":     results,
		"count":       len(results),
		"time":        time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}
