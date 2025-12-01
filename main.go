package main

import (
	"fmt"
	"log"
	"net/http"

	"sort"

	"strings"
	"sync"
	"time"
)

// StreetRecord represents a single street entry from the CSV
type StreetRecord struct {
	WOJ      int    `json:"woj"`
	POW      int    `json:"pow"`
	GMI      int    `json:"gmi"`
	RODZGMI  int    `json:"rodz_gmi"`
	SYM      int    `json:"sym"`
	SYMUL    int    `json:"sym_ul"`
	CECHA    string `json:"cecha"`
	NAZWA1   string `json:"nazwa_1"`
	NAZWA2   string `json:"nazwa_2"`
	FullName string `json:"full_name"`
}

// CityRecord represents a single locality/city entry from the SIMC CSV
type CityRecord struct {
	WOJ     int    `json:"woj"`
	POW     int    `json:"pow"`
	GMI     int    `json:"gmi"`
	RODZGMI int    `json:"rodz_gmi"`
	RM      int    `json:"rm"`
	MZ      int    `json:"mz"`
	NAZWA   string `json:"nazwa"`
	SYM     int    `json:"sym"`
	SYMPOD  int    `json:"sympod"`
}

// AutocompleteService manages the in-memory street and city indexes
type AutocompleteService struct {
	streets []StreetRecord
	cities  []CityRecord
	mu      sync.RWMutex
}

// NewAutocompleteService creates a new autocomplete service
func NewAutocompleteService() *AutocompleteService {
	return &AutocompleteService{
		streets: make([]StreetRecord, 0),
		cities:  make([]CityRecord, 0),
	}
}

// SearchCities performs autocomplete search on city names (NAZWA) with optional filtering
func (s *AutocompleteService) SearchCities(query string, woj, pow, gmi int, limit int) []CityRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]CityRecord, 0, limit)
	seen := make(map[string]bool)

	for _, city := range s.cities {
		if len(results) >= limit {
			break
		}

		// Apply administrative unit filters if specified (0 means no filter)
		if woj > 0 && city.WOJ != woj {
			continue
		}
		if pow > 0 && city.POW != pow {
			continue
		}
		if gmi > 0 && city.GMI != gmi {
			continue
		}

		// Search in NAZWA (city name)
		nazwaLower := strings.ToLower(city.NAZWA)

		// If query is empty, match all (with filters applied above)
		matchesQuery := query == "" || strings.HasPrefix(nazwaLower, query) || strings.Contains(nazwaLower, query)

		if matchesQuery {
			// Deduplicate by name + administrative codes
			key := fmt.Sprintf("%s-%d-%d-%d", city.NAZWA, city.WOJ, city.POW, city.GMI)
			if !seen[key] {
				results = append(results, city)
				seen[key] = true
			}
		}
	}

	// Sort results: prefix matches first, then contains matches, then alphabetically
	sort.Slice(results, func(i, j int) bool {
		if query == "" {
			return results[i].NAZWA < results[j].NAZWA
		}

		nazwaI := strings.ToLower(results[i].NAZWA)
		nazwaJ := strings.ToLower(results[j].NAZWA)

		prefixI := strings.HasPrefix(nazwaI, query)
		prefixJ := strings.HasPrefix(nazwaJ, query)

		if prefixI && !prefixJ {
			return true
		}
		if !prefixI && prefixJ {
			return false
		}

		return results[i].NAZWA < results[j].NAZWA
	})

	return results
}

// GetGMIForStreet returns unique GMI codes where the exact street name exists
func (s *AutocompleteService) GetGMIForStreet(streetName string) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	streetName = strings.TrimSpace(streetName)
	if streetName == "" {
		return []map[string]interface{}{}
	}

	streetNameLower := strings.ToLower(streetName)
	seen := make(map[string]bool)
	var results []map[string]interface{}

	for _, street := range s.streets {
		nazwa1Lower := strings.ToLower(street.NAZWA1)

		// Exact match on NAZWA_1
		if nazwa1Lower == streetNameLower {
			// Create unique key for WOJ-POW-GMI combination
			key := fmt.Sprintf("%d-%d-%d", street.WOJ, street.POW, street.GMI)

			if !seen[key] {
				results = append(results, map[string]interface{}{
					"woj": street.WOJ,
					"pow": street.POW,
					"gmi": street.GMI,
				})
				seen[key] = true
			}
		}
	}

	// Sort by WOJ, POW, GMI
	sort.Slice(results, func(i, j int) bool {
		if results[i]["woj"].(int) != results[j]["woj"].(int) {
			return results[i]["woj"].(int) < results[j]["woj"].(int)
		}
		if results[i]["pow"].(int) != results[j]["pow"].(int) {
			return results[i]["pow"].(int) < results[j]["pow"].(int)
		}
		return results[i]["gmi"].(int) < results[j]["gmi"].(int)
	})

	return results
}

// SearchStreets performs autocomplete search on NAZWA_1 (street name)
func (s *AutocompleteService) SearchStreets(query string, limit int) []StreetRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" {
		return []StreetRecord{}
	}

	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]StreetRecord, 0, limit)
	seen := make(map[string]bool)

	for _, street := range s.streets {
		if len(results) >= limit {
			break
		}

		// Search in NAZWA_1 (main street name)
		nazwa1Lower := strings.ToLower(street.NAZWA1)

		if strings.HasPrefix(nazwa1Lower, query) || strings.Contains(nazwa1Lower, query) {
			// Deduplicate by full name
			if !seen[street.FullName] {
				results = append(results, street)
				seen[street.FullName] = true
			}
		}
	}

	// Sort results: prefix matches first, then contains matches
	sort.Slice(results, func(i, j int) bool {
		nazwa1i := strings.ToLower(results[i].NAZWA1)
		nazwa1j := strings.ToLower(results[j].NAZWA1)

		prefixI := strings.HasPrefix(nazwa1i, query)
		prefixJ := strings.HasPrefix(nazwa1j, query)

		if prefixI && !prefixJ {
			return true
		}
		if !prefixI && prefixJ {
			return false
		}

		return results[i].NAZWA1 < results[j].NAZWA1
	})

	return results
}

// AutocompleteResponse is the JSON response structure for streets
type AutocompleteResponse struct {
	Query   string         `json:"query"`
	Results []StreetRecord `json:"results"`
	Count   int            `json:"count"`
	Time    string         `json:"time"`
}

// CityAutocompleteResponse is the JSON response structure for cities
type CityAutocompleteResponse struct {
	Query   string         `json:"query"`
	Filters map[string]int `json:"filters,omitempty"`
	Results []CityRecord   `json:"results"`
	Count   int            `json:"count"`
	Time    string         `json:"time"`
}

var service *AutocompleteService

func main() {
	// Initialize service
	service = NewAutocompleteService()

	// Load street data
	streetFile := "data/ULIC_Adresowy_2025-12-01.csv"
	log.Printf("Loading street data from %s...", streetFile)
	startTime := time.Now()
	if err := service.LoadCSV(streetFile); err != nil {
		log.Fatalf("Failed to load streets CSV: %v", err)
	}
	log.Printf("Streets loaded in %v", time.Since(startTime))

	// Load city data
	cityFile := "data/SIMC_Adresowy_2025-12-01.csv"
	log.Printf("Loading city data from %s...", cityFile)
	startTime = time.Now()
	if err := service.LoadCitiesCSV(cityFile); err != nil {
		log.Fatalf("Failed to load cities CSV: %v", err)
	}
	log.Printf("Cities loaded in %v", time.Since(startTime))

	// Setup HTTP routes
	http.HandleFunc("/streets", streetsHandler)
	http.HandleFunc("/streets/gmi", streetGMIHandler)
	http.HandleFunc("/cities", citiesHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/demo", demoHandler)
	http.HandleFunc("/", rootHandler)

	// Start server
	port := ":8081"
	log.Printf("Starting server on http://localhost%s", port)
	log.Printf("Try: http://localhost%s/streets?q=Chopina", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
