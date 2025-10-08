# 🌍 Boundary Downloader

A console app for downloading administrative boundary data from OpenStreetMap via Overpass API and converting to GeoJSON.

## Features

- 🔍 Search and select from 200+ countries
- 📊 Download boundaries for admin levels 2-10 (countries to city blocks)
- 🎨 Interactive UI with arrow key navigation
-  View downloaded files with stats (size, features, date)
- 🌐 Wikidata-based queries for reliable results

## Installation

```bash
git clone git@github.com:albertomario/geojson-boundary-fetcher.git
cd boundary-downloader
go build
```

## Usage

```bash
./boundary-fetcher
```

### Menu Options

1. **View Downloaded Boundaries** - Browse downloaded files in a table
2. **Fetch New Boundary** - 3-step download process:
   - Search & select country
   - Choose admin level (2-10)
   - Confirm and download
3. **Quit**

### Administrative Levels

| Level | Description |
|-------|-------------|
| 2 | Countries |
| 3 | States/Provinces |
| 4 | Regions/Counties |
| 5 | Districts |
| 6 | Municipalities |
| 7 | Wards/Neighborhoods |
| 8 | Villages/Sub-districts |
| 9 | Quarters/Hamlets |
| 10 | City blocks |

### Keyboard Controls

- **↑/↓** Navigate
- **Enter** Select/Confirm
- **ESC** Cancel/Back

## Output

Files saved to: `geojson/{COUNTRY_CODE}/{ADMIN_LEVEL}/boundary.geojson`

Example: `geojson/Q218/8/boundary.geojson` (Romania, level 8)

## Dependencies

- Go 1.25+
- `github.com/eiannone/keyboard`
- `github.com/paulmach/go.geojson`
- `github.com/jedib0t/go-pretty/v6/table`

## Data Source

- OpenStreetMap contributors
- Overpass API
- Wikidata for country codes

---

Made with ❤️ for the mapping community
