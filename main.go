package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
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

// LoadCSV loads the street data from CSV file into memory
func (s *AutocompleteService) LoadCSV(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Use manual line-by-line parsing due to CSV data quality issues
	// The file has unescaped quotes that confuse the standard CSV reader
	scanner := strings.NewReader("")
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	scanner = strings.NewReader(string(data))

	s.streets = make([]StreetRecord, 0, 300000)
	lineNum := 0
	skipped := 0

	// Skip header line
	_, _ = readLine(scanner)
	lineNum++

	for {
		line, err := readLine(scanner)
		if err != nil {
			break
		}
		lineNum++

		// Split by semicolon
		fields := strings.Split(line, ";")

		// Validate record has exactly 10 fields
		if len(fields) != 10 {
			skipped++
			continue
		}

		// Clean up fields by removing any quotes
		for i := range fields {
			fields[i] = strings.Trim(strings.TrimSpace(fields[i]), "\"")
		}

		// Validate essential fields are not empty
		if fields[7] == "" { // NAZWA_1 must not be empty
			skipped++
			continue
		}

		// Parse integer fields
		woj, _ := strconv.Atoi(fields[0])
		pow, _ := strconv.Atoi(fields[1])
		gmi, _ := strconv.Atoi(fields[2])
		rodzgmi, _ := strconv.Atoi(fields[3])
		sym, _ := strconv.Atoi(fields[4])
		symul, _ := strconv.Atoi(fields[5])

		street := StreetRecord{
			WOJ:     woj,
			POW:     pow,
			GMI:     gmi,
			RODZGMI: rodzgmi,
			SYM:     sym,
			SYMUL:   symul,
			CECHA:   fields[6],
			NAZWA1:  fields[7],
			NAZWA2:  fields[8],
		}

		// Build full name for display
		if street.NAZWA2 != "" {
			street.FullName = fmt.Sprintf("%s %s %s", street.CECHA, street.NAZWA1, street.NAZWA2)
		} else {
			street.FullName = fmt.Sprintf("%s %s", street.CECHA, street.NAZWA1)
		}

		s.streets = append(s.streets, street)
	}

	log.Printf("Loaded %d street records from %s (skipped %d malformed records)", len(s.streets), filename, skipped)
	return nil
}

// LoadCitiesCSV loads the city/locality data from SIMC CSV file into memory
func (s *AutocompleteService) LoadCitiesCSV(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	scanner := strings.NewReader(string(data))

	s.cities = make([]CityRecord, 0, 100000)
	lineNum := 0
	skipped := 0

	// Skip header line
	_, _ = readLine(scanner)
	lineNum++

	for {
		line, err := readLine(scanner)
		if err != nil {
			break
		}
		lineNum++

		// Split by semicolon
		fields := strings.Split(line, ";")

		// Validate record has exactly 10 fields
		if len(fields) != 10 {
			skipped++
			continue
		}

		// Clean up fields by removing any quotes
		for i := range fields {
			fields[i] = strings.Trim(strings.TrimSpace(fields[i]), "\"")
		}

		// Validate essential fields are not empty
		if fields[6] == "" { // NAZWA must not be empty
			skipped++
			continue
		}

		// Parse integer fields
		woj, _ := strconv.Atoi(fields[0])
		pow, _ := strconv.Atoi(fields[1])
		gmi, _ := strconv.Atoi(fields[2])
		rodzgmi, _ := strconv.Atoi(fields[3])
		rm, _ := strconv.Atoi(fields[4])
		mz, _ := strconv.Atoi(fields[5])
		sym, _ := strconv.Atoi(fields[7])
		sympod, _ := strconv.Atoi(fields[8])

		city := CityRecord{
			WOJ:     woj,
			POW:     pow,
			GMI:     gmi,
			RODZGMI: rodzgmi,
			RM:      rm,
			MZ:      mz,
			NAZWA:   fields[6],
			SYM:     sym,
			SYMPOD:  sympod,
		}

		s.cities = append(s.cities, city)
	}

	log.Printf("Loaded %d city records from %s (skipped %d malformed records)", len(s.cities), filename, skipped)
	return nil
}

// readLine reads a single line from a strings.Reader
func readLine(r *strings.Reader) (string, error) {
	var line strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil {
			if line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
		if b == '\n' {
			return line.String(), nil
		}
		if b != '\r' { // Skip carriage return
			line.WriteByte(b)
		}
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

// Search performs autocomplete search on NAZWA_1 (street name)
func (s *AutocompleteService) Search(query string, limit int) []StreetRecord {
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
	Query   string       `json:"query"`
	Filters map[string]int `json:"filters,omitempty"`
	Results []CityRecord `json:"results"`
	Count   int          `json:"count"`
	Time    string       `json:"time"`
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
	http.HandleFunc("/streets", autocompleteHandler)
	http.HandleFunc("/streets/gmi", streetGMIHandler)
	http.HandleFunc("/cities", citiesHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", rootHandler)

	// Start server
	port := ":8080"
	log.Printf("Starting server on http://localhost%s", port)
	log.Printf("Try: http://localhost%s/streets?q=Chopina", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
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
	results := service.Search(query, limit)

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"streets": fmt.Sprintf("%d", len(service.streets)),
		"cities":  fmt.Sprintf("%d", len(service.cities)),
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Street Autocomplete API</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .endpoint { background: #f4f4f4; padding: 10px; margin: 10px 0; border-radius: 5px; }
        code { background: #e0e0e0; padding: 2px 5px; border-radius: 3px; }
        input { padding: 10px; width: 300px; font-size: 16px; }
        #results { margin-top: 20px; }
        .result { padding: 8px; margin: 5px 0; background: #f9f9f9; border-left: 3px solid #4CAF50; }
    </style>
</head>
<body>
    <h1>Polish Street Autocomplete API</h1>
    <p>Try the autocomplete:</p>
    <input type="text" id="search" placeholder="Type street name..." onkeyup="search()">
    <div id="results"></div>

    <h2>API Endpoints</h2>
    <div class="endpoint">
        <strong>GET /streets?q={query}</strong><br>
        Search for streets by name<br>
        Example: <a href="/streets?q=Chopina">/streets?q=Chopina</a>
    </div>
    <div class="endpoint">
        <strong>GET /streets/gmi?name={exact_street_name}</strong><br>
        Get list of GMI codes where an exact street name exists<br>
        Example: <a href="/streets/gmi?name=Sportowa">/streets/gmi?name=Sportowa</a>
    </div>
    <div class="endpoint">
        <strong>GET /cities?q={query}&woj={woj}&pow={pow}&gmi={gmi}</strong><br>
        Search for cities by name with optional filters<br>
        Examples:<br>
        - <a href="/cities?q=Warszawa">/cities?q=Warszawa</a><br>
        - <a href="/cities?q=Krak&woj=12">/cities?q=Krak&woj=12</a> (filter by województwo)<br>
        - <a href="/cities?woj=14&pow=32">/cities?woj=14&pow=32</a> (all cities in powiat)
    </div>
    <div class="endpoint">
        <strong>GET /health</strong><br>
        Example: <a href="/health">/health</a>
    </div>

    <script>
        let timeout = null;
        function search() {
            clearTimeout(timeout);
            const query = document.getElementById('search').value;

            if (query.length < 2) {
                document.getElementById('results').innerHTML = '';
                return;
            }

            timeout = setTimeout(() => {
                fetch('/streets?q=' + encodeURIComponent(query))
                    .then(r => r.json())
                    .then(data => {
                        const html = data.results.map(r =>
                            '<div class="result">' + r.full_name + '</div>'
                        ).join('');
                        document.getElementById('results').innerHTML =
                            '<p>Found ' + data.count + ' results in ' + data.time + '</p>' + html;
                    });
            }, 200);
        }
    </script>
</body>
</html>`
	w.Write([]byte(html))
}
