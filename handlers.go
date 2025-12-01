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
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"streets": len(service.streets),
		"cities":  len(service.cities),
	})
}

func streetsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get query parameter
	query := r.URL.Query().Get("q")

	// Default limit
	limit := 10

	// Search
	results := service.SearchStreets(query, limit)

	// Check if this is an HTMX request
	isHTMX := r.Header.Get("HX-Request") == "true"

	if isHTMX {
		// Return HTML fragment for HTMX
		w.Header().Set("Content-Type", "text/html")

		if query == "" || len(results) == 0 {
			if query == "" {
				fmt.Fprint(w, `<div class="empty-state">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
					</svg>
					<p>Wpisz co najmniej 2 znaki</p>
				</div>`)
			} else {
				fmt.Fprint(w, `<div class="no-results">Nie znaleziono wyników</div>`)
			}
			return
		}

		// Render results as HTML
		fmt.Fprintf(w, `<div class="results-info">Znaleziono %d wyników w %s</div>`, len(results), time.Since(startTime).String())
		fmt.Fprint(w, `<ul class="results-list">`)
		for _, result := range results {
			fmt.Fprintf(w, `<li class="result-item">
				<div class="result-name">%s</div>
			</li>`,
				result.FullName)
		}
		fmt.Fprint(w, `</ul>`)
		return
	}

	// Return JSON for regular API requests
	w.Header().Set("Content-Type", "application/json")

	if query == "" {
		json.NewEncoder(w).Encode(AutocompleteResponse{
			Query:   "",
			Results: []StreetRecord{},
			Count:   0,
			Time:    time.Since(startTime).String(),
		})
		return
	}

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

	// Check if this is an HTMX request
	isHTMX := r.Header.Get("HX-Request") == "true"

	if isHTMX {
		// Return HTML fragment for HTMX
		w.Header().Set("Content-Type", "text/html")

		if query == "" || len(results) == 0 {
			if query == "" {
				fmt.Fprint(w, `<div class="empty-state">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
					</svg>
					<p>Wpisz co najmniej 2 znaki</p>
				</div>`)
			} else {
				fmt.Fprint(w, `<div class="no-results">Nie znaleziono wyników</div>`)
			}
			return
		}

		// Deduplicate by city name for display
		seen := make(map[string]bool)
		uniqueResults := []CityRecord{}
		for _, result := range results {
			if !seen[result.NAZWA] {
				uniqueResults = append(uniqueResults, result)
				seen[result.NAZWA] = true
			}
		}

		// Render results as HTML
		fmt.Fprintf(w, `<div class="results-info">Znaleziono %d wyników w %s</div>`, len(uniqueResults), time.Since(startTime).String())
		fmt.Fprint(w, `<ul class="results-list">`)
		for _, result := range uniqueResults {
			fmt.Fprintf(w, `<li class="result-item">
				<div class="result-name">%s</div>
			</li>`,
				result.NAZWA)
		}
		fmt.Fprint(w, `</ul>`)
		return
	}

	// Return JSON for regular API requests
	w.Header().Set("Content-Type", "application/json")

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
		json.NewEncoder(w).Encode(map[string]any{
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
	response := map[string]any{
		"street_name": streetName,
		"results":     results,
		"count":       len(results),
		"time":        time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}
