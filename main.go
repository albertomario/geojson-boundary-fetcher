package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/jedib0t/go-pretty/v6/table"
)

type Country struct {
	Country      string `json:"country"`
	CountryLabel string `json:"countryLabel"`
	CountryID    string `json:"countryID"`
}

type DownloadedBoundary struct {
	CountryCode  string
	CountryLabel string
	AdminLevel   int
	FilePath     string
	FileSize     int64
	Features     int
	ModTime      string
}

func main() {
	fmt.Println("Boundary Downloader - Console App")
	fmt.Println("==================================")

	// Load countries
	data, err := os.ReadFile("country-list.json")
	if err != nil {
		fmt.Printf("❌ Error loading country list: %v\n", err)
		os.Exit(1)
	}

	var countries []Country
	err = json.Unmarshal(data, &countries)
	if err != nil {
		fmt.Printf("❌ Error parsing country list: %v\n", err)
		os.Exit(1)
	}

	// Main loop
	for {
		action := showMainMenu()
		if action == "quit" {
			fmt.Println("Goodbye!")
			os.Exit(0)
		}

		if action == "view" {
			showDownloaded(countries)
		} else if action == "fetch" {
			fetchNewBoundary(countries)
		}
	}
}

func showMainMenu() string {
	if err := keyboard.Open(); err != nil {
		fmt.Printf("Failed to open keyboard: %v\n", err)
		return "quit"
	}
	defer keyboard.Close()

	options := []string{"View Downloaded Boundaries", "Fetch New Boundary", "Quit"}
	selectedIndex := 0

	for {
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("Boundary Downloader - Main Menu")
		fmt.Println("Use ↑/↓ arrow keys to navigate, Enter to select")
		fmt.Println("========================================")

		for i, option := range options {
			if i == selectedIndex {
				fmt.Printf("→ %s\n", option)
			} else {
				fmt.Printf("  %s\n", option)
			}
		}

		char, key, err := keyboard.GetKey()
		if err != nil {
			return "quit"
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
		case keyboard.KeyArrowDown:
			if selectedIndex < len(options)-1 {
				selectedIndex++
			}
		case keyboard.KeyEnter:
			switch selectedIndex {
			case 0:
				return "view"
			case 1:
				return "fetch"
			case 2:
				return "quit"
			}
		case keyboard.KeyEsc:
			return "quit"
		}

		if char == 'q' || char == 'Q' {
			return "quit"
		}
	}
}

func showDownloaded(countries []Country) {
	downloaded := getDownloadedBoundaries(countries)

	if len(downloaded) == 0 {
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("No downloaded boundaries found.")
		fmt.Println("\nPress any key to return to main menu...")
		keyboard.Open()
		keyboard.GetKey()
		keyboard.Close()
		return
	}

	// Group by country
	type CountryData struct {
		Code   string
		Label  string
		Levels []DownloadedBoundary
	}

	countryMap := make(map[string]*CountryData)

	for _, dl := range downloaded {
		if _, exists := countryMap[dl.CountryCode]; !exists {
			countryMap[dl.CountryCode] = &CountryData{
				Code:   dl.CountryCode,
				Label:  dl.CountryLabel,
				Levels: []DownloadedBoundary{},
			}
		}
		countryMap[dl.CountryCode].Levels = append(countryMap[dl.CountryCode].Levels, dl)
	}

	// Sort levels for each country
	for _, cd := range countryMap {
		sort.Slice(cd.Levels, func(i, j int) bool {
			return cd.Levels[i].AdminLevel < cd.Levels[j].AdminLevel
		})
	}

	// Create sorted list of countries
	var countryData []*CountryData
	for _, cd := range countryMap {
		countryData = append(countryData, cd)
	}
	sort.Slice(countryData, func(i, j int) bool {
		return countryData[i].Label < countryData[j].Label
	})

	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("Downloaded Boundaries")
	fmt.Println()

	// Create table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Country", "Code", "Level", "Features", "Size", "Downloaded"})

	for _, cd := range countryData {
		for i, dl := range cd.Levels {
			countryName := ""
			code := ""
			if i == 0 {
				countryName = cd.Label
				code = cd.Code
			}

			sizeStr := formatFileSize(dl.FileSize)

			t.AppendRow(table.Row{
				countryName,
				code,
				dl.AdminLevel,
				dl.Features,
				sizeStr,
				dl.ModTime,
			})
		}
		t.AppendSeparator()
	}

	t.SetStyle(table.StyleColoredBright)
	t.Style().Options.SeparateRows = false
	t.Render()

	fmt.Println("\nPress any key to return to main menu...")
	keyboard.Open()
	keyboard.GetKey()
	keyboard.Close()
}

func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func getDownloadedBoundaries(countries []Country) []DownloadedBoundary {
	var downloaded []DownloadedBoundary

	// Create country map for lookup
	countryMap := make(map[string]string)
	for _, c := range countries {
		countryMap[getQNumber(c.CountryID)] = c.CountryLabel
	}

	// Scan geojson directory
	if _, err := os.Stat("geojson"); os.IsNotExist(err) {
		return downloaded
	}

	entries, err := os.ReadDir("geojson")
	if err != nil {
		return downloaded
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		countryCode := entry.Name()
		countryLabel := countryMap[countryCode]
		if countryLabel == "" {
			countryLabel = countryCode
		}

		// Check for admin levels
		countryPath := filepath.Join("geojson", countryCode)
		levelEntries, err := os.ReadDir(countryPath)
		if err != nil {
			continue
		}

		for _, levelEntry := range levelEntries {
			if !levelEntry.IsDir() {
				continue
			}

			level, err := strconv.Atoi(levelEntry.Name())
			if err != nil {
				continue
			}

			filePath := filepath.Join(countryPath, levelEntry.Name(), "boundary.geojson")
			if fileInfo, err := os.Stat(filePath); err == nil {
				// Read file to count features
				features := 0
				if data, err := os.ReadFile(filePath); err == nil {
					var fc map[string]interface{}
					if err := json.Unmarshal(data, &fc); err == nil {
						if featuresArray, ok := fc["features"].([]interface{}); ok {
							features = len(featuresArray)
						}
					}
				}

				downloaded = append(downloaded, DownloadedBoundary{
					CountryCode:  countryCode,
					CountryLabel: countryLabel,
					AdminLevel:   level,
					FilePath:     filePath,
					FileSize:     fileInfo.Size(),
					Features:     features,
					ModTime:      fileInfo.ModTime().Format("2006-01-02"),
				})
			}
		}
	}

	// Sort by country label, then admin level
	sort.Slice(downloaded, func(i, j int) bool {
		if downloaded[i].CountryLabel == downloaded[j].CountryLabel {
			return downloaded[i].AdminLevel < downloaded[j].AdminLevel
		}
		return downloaded[i].CountryLabel < downloaded[j].CountryLabel
	})

	return downloaded
}

func fetchNewBoundary(countries []Country) {
	// Step 1: Select country
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   Fetch New Boundary - Step 1 of 3  ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	selected := selectCountry(countries)
	if selected == nil {
		showCancelMessage("Country selection cancelled")
		return
	}

	// Step 2: Get admin level
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   Fetch New Boundary - Step 2 of 3  ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Selected Country: %s (%s)\n", selected.CountryLabel, getQNumber(selected.CountryID))
	fmt.Println()

	level := getAdminLevel()
	if level == -1 {
		showCancelMessage("Admin level selection cancelled")
		return
	}

	// Step 3: Confirm and fetch
	countryCode := getQNumber(selected.CountryID)

	// Check if already exists
	existingFile := filepath.Join("geojson", countryCode, strconv.Itoa(level), "boundary.geojson")
	alreadyExists := false
	if _, err := os.Stat(existingFile); err == nil {
		alreadyExists = true
	}

	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   Fetch New Boundary - Step 3 of 3  ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📋 Summary:")
	fmt.Println("─────────────────────────────────────")
	fmt.Printf("  Country:     %s\n", selected.CountryLabel)
	fmt.Printf("  Code:        %s\n", countryCode)
	fmt.Printf("  Admin Level: %d\n", level)
	fmt.Println()

	if alreadyExists {
		fmt.Println("⚠️  Warning: This boundary already exists and will be overwritten!")
		fmt.Println()
	}

	if !confirmAction("Do you want to proceed with the download?") {
		showCancelMessage("Download cancelled")
		return
	}

	// Fetch
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║        Downloading Boundary         ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Country: %s (%s)\n", selected.CountryLabel, countryCode)
	fmt.Printf("Level:   %d\n", level)
	fmt.Println()
	fmt.Println("Please wait...")
	fmt.Println()

	err := fetchBoundary(level, selected.CountryLabel, countryCode)
	if err != nil {
		fmt.Println()
		fmt.Printf("❌ Error: %v\n", err)
		fmt.Println("\nPress any key to continue...")
		keyboard.Open()
		keyboard.GetKey()
		keyboard.Close()
		return
	}

	fmt.Println()
	fmt.Println("✅ Success! Boundary downloaded successfully.")
	fmt.Println("\nPress any key to continue...")
	keyboard.Open()
	keyboard.GetKey()
	keyboard.Close()
}

func showCancelMessage(message string) {
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║           Cancelled                  ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("ℹ️  %s\n", message)
	fmt.Println("\nPress any key to return to main menu...")
	keyboard.Open()
	keyboard.GetKey()
	keyboard.Close()
}

func confirmAction(message string) bool {
	if err := keyboard.Open(); err != nil {
		return false
	}
	defer keyboard.Close()

	options := []string{"Yes, proceed", "No, cancel"}
	selectedIndex := 0

	for {
		fmt.Println(message)
		fmt.Println()
		fmt.Println("Use ↑/↓ arrow keys to navigate, Enter to select")
		fmt.Println()

		for i, option := range options {
			if i == selectedIndex {
				if i == 0 {
					fmt.Printf("→ ✓ %s\n", option)
				} else {
					fmt.Printf("→ ✗ %s\n", option)
				}
			} else {
				if i == 0 {
					fmt.Printf("  ✓ %s\n", option)
				} else {
					fmt.Printf("  ✗ %s\n", option)
				}
			}
		}

		_, key, err := keyboard.GetKey()
		if err != nil {
			return false
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
			// Clear lines and redraw
			fmt.Print("\033[5A\033[J")
		case keyboard.KeyArrowDown:
			if selectedIndex < len(options)-1 {
				selectedIndex++
			}
			// Clear lines and redraw
			fmt.Print("\033[5A\033[J")
		case keyboard.KeyEnter:
			return selectedIndex == 0
		case keyboard.KeyEsc:
			return false
		}
	}
}

func selectCountry(countries []Country) *Country {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("🔍 Search for a country")
		fmt.Println("─────────────────────────────────────")
		fmt.Print("Enter search term (or type 'cancel' to go back): ")
		scanner.Scan()
		filter := strings.TrimSpace(scanner.Text())
		if filter == "cancel" || filter == "quit" {
			return nil
		}

		if filter == "" {
			fmt.Println("⚠️  Please enter a search term.")
			fmt.Println()
			continue
		}

		filtered := []Country{}
		for _, c := range countries {
			if strings.Contains(strings.ToLower(c.CountryLabel), strings.ToLower(filter)) {
				filtered = append(filtered, c)
			}
		}

		if len(filtered) == 0 {
			fmt.Println("❌ No countries found. Try again.")
			fmt.Println()
			continue
		}

		fmt.Printf("\n✓ Found %d matching countr", len(filtered))
		if len(filtered) == 1 {
			fmt.Println("y")
		} else {
			fmt.Println("ies")
		}
		fmt.Println()

		selected := navigateList(filtered)
		if selected != nil {
			return selected
		}

		// If user cancelled navigation, go back to search
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("╔══════════════════════════════════════╗")
		fmt.Println("║   Fetch New Boundary - Step 1 of 3  ║")
		fmt.Println("╚══════════════════════════════════════╝")
		fmt.Println()
	}
}

func navigateList(countries []Country) *Country {
	if err := keyboard.Open(); err != nil {
		fmt.Printf("❌ Failed to open keyboard: %v\n", err)
		return nil
	}
	defer keyboard.Close()

	selectedIndex := 0

	for {
		// Clear screen and display list
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("╔══════════════════════════════════════╗")
		fmt.Println("║        Select Country                ║")
		fmt.Println("╚══════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("Use ↑/↓ arrow keys to navigate")
		fmt.Println("Press Enter to select, ESC to cancel")
		fmt.Println("─────────────────────────────────────")
		fmt.Println()

		// Show limited view with scroll
		displayStart := 0
		displayEnd := len(countries)
		maxDisplay := 15 // Show max 15 items at a time

		if len(countries) > maxDisplay {
			if selectedIndex > maxDisplay/2 {
				displayStart = selectedIndex - maxDisplay/2
			}
			displayEnd = displayStart + maxDisplay
			if displayEnd > len(countries) {
				displayEnd = len(countries)
				displayStart = displayEnd - maxDisplay
				if displayStart < 0 {
					displayStart = 0
				}
			}
		}

		if displayStart > 0 {
			fmt.Printf("    ... (%d more above)\n", displayStart)
		}

		for i := displayStart; i < displayEnd; i++ {
			c := countries[i]
			if i == selectedIndex {
				fmt.Printf("  → %s (%s)\n", c.CountryLabel, getQNumber(c.CountryID))
			} else {
				fmt.Printf("    %s (%s)\n", c.CountryLabel, getQNumber(c.CountryID))
			}
		}

		if displayEnd < len(countries) {
			fmt.Printf("    ... (%d more below)\n", len(countries)-displayEnd)
		}

		char, key, err := keyboard.GetKey()
		if err != nil {
			fmt.Printf("❌ Error reading key: %v\n", err)
			return nil
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
		case keyboard.KeyArrowDown:
			if selectedIndex < len(countries)-1 {
				selectedIndex++
			}
		case keyboard.KeyEnter:
			return &countries[selectedIndex]
		case keyboard.KeyEsc:
			return nil
		}

		if char == 'q' || char == 'Q' {
			return nil
		}
	}
}

func getAdminLevel() int {
	levelInfo := map[int]string{
		2:  "Countries",
		3:  "First-level (states, provinces)",
		4:  "Second-level (regions, counties)",
		5:  "Third-level (districts)",
		6:  "Fourth-level (municipalities)",
		7:  "Fifth-level (wards, neighborhoods)",
		8:  "Sixth-level (villages, sub-districts)",
		9:  "Seventh-level (quarters, hamlets)",
		10: "Eighth-level (city blocks)",
	}

	if err := keyboard.Open(); err != nil {
		fmt.Printf("❌ Failed to open keyboard: %v\n", err)
		return 8
	}
	defer keyboard.Close()

	selectedLevel := 8
	levels := []int{2, 3, 4, 5, 6, 7, 8, 9, 10}
	selectedIndex := 6 // Default to level 8

	for {
		// Clear screen and display levels
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("╔══════════════════════════════════════╗")
		fmt.Println("║    Select Administrative Level       ║")
		fmt.Println("╚══════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("Use ↑/↓ arrow keys to navigate")
		fmt.Println("Press Enter to select, ESC to cancel")
		fmt.Println("─────────────────────────────────────")
		fmt.Println()

		for i, level := range levels {
			desc := levelInfo[level]
			if i == selectedIndex {
				fmt.Printf("  → Level %d: %s\n", level, desc)
			} else {
				fmt.Printf("    Level %d: %s\n", level, desc)
			}
		}

		char, key, err := keyboard.GetKey()
		if err != nil {
			fmt.Printf("❌ Error reading key: %v\n", err)
			return 8
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
		case keyboard.KeyArrowDown:
			if selectedIndex < len(levels)-1 {
				selectedIndex++
			}
		case keyboard.KeyEnter:
			selectedLevel = levels[selectedIndex]
			return selectedLevel
		case keyboard.KeyEsc:
			return -1 // Signal cancellation
		}

		if char == 'q' || char == 'Q' {
			return -1 // Signal cancellation
		}
	}
}

func getQNumber(countryID string) string {
	parts := strings.Split(countryID, "/")
	return parts[len(parts)-1]
}
