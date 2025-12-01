# Polish Address Autocomplete API

Fast, in-memory autocomplete API for Polish streets and cities built with Go.

## Features

- **Street Search** - Autocomplete search across 305K+ Polish streets
- **City Search** - Search 95K+ localities with administrative filters
- **GMI Lookup** - Find all municipalities (GMI) where a street exists
- **Fast** - In-memory index, <50ms response times
- **Simple** - Single binary, no database required

## Quick Start

```bash
# Run the server
go run main.go

# Server starts on http://localhost:8080
```

## API Endpoints

### Search Streets
```bash
GET /streets?q={query}

curl "http://localhost:8080/streets?q=Chopina"
```

### Get GMI Codes for Street
```bash
GET /streets/gmi?name={exact_street_name}

curl "http://localhost:8080/streets/gmi?name=Sportowa"
```

### Search Cities
```bash
GET /cities?q={query}&woj={woj}&pow={pow}&gmi={gmi}

# Search by name
curl "http://localhost:8080/cities?q=Warszawa"

# Filter by województwo
curl "http://localhost:8080/cities?q=Krak&woj=12"

# Get all cities in powiat
curl "http://localhost:8080/cities?woj=14&pow=32"
```

### Health Check
```bash
GET /health

curl "http://localhost:8080/health"
```

### Web UI
Open http://localhost:8080/ in your browser for an interactive demo.

## Data Files

The API uses official Polish address registry data:
- `data/ULIC_Adresowy_*.csv` - Street data (~305K records)
- `data/SIMC_Adresowy_*.csv` - City/locality data (~95K records)

## Building

```bash
# Build binary
go build -o autocomplete.exe main.go

# Run binary
./autocomplete.exe
```

## Technical Details

- **Language**: Go 1.24.5
- **Architecture**: Single HTTP server with in-memory index
- **Startup Time**: ~300ms to load all data
- **Memory Usage**: ~100MB for full dataset
- **Concurrency**: Thread-safe with RWMutex

## Response Format

All endpoints return JSON with timing information:

```json
{
  "query": "Chopina",
  "results": [...],
  "count": 10,
  "time": "15.2ms"
}
```

## License

This project uses public domain address data from the Polish government address registry (TERYT).
