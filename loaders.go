package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

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
