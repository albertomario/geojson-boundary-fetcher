package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	geojson "github.com/paulmach/go.geojson"
)

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Member struct {
	Type     string  `json:"type"`
	Ref      int64   `json:"ref"`
	Role     string  `json:"role"`
	Geometry []Point `json:"geometry"`
}

type Element struct {
	Type    string            `json:"type"`
	Id      int64             `json:"id"`
	Tags    map[string]string `json:"tags"`
	Members []Member          `json:"members"`
}

type OSMResponse struct {
	Elements []Element `json:"elements"`
}

func fetchBoundary(adminLevel int, countryName, countryCode string) error {
	fmt.Printf("Fetching administrative boundaries for %s (%s) at level %d...\n", countryName, countryCode, adminLevel)

	// Use Wikidata ID to query boundaries - more reliable than name matching
	// Query for all administrative boundaries at the specified level within the country
	query := fmt.Sprintf(`[out:json][timeout:90];
		relation["wikidata"="%s"];
		map_to_area->.country;
		(
			relation(area.country)["boundary"="administrative"]["admin_level"="%d"];
		);
		out geom;`, countryCode, adminLevel)

	encodedQuery := url.QueryEscape(query)
	apiURL := "https://overpass-api.de/api/interpreter?data=" + encodedQuery

	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var osmResp OSMResponse
	err = json.Unmarshal(data, &osmResp)
	if err != nil {
		return err
	}

	fmt.Printf("Received %d boundary elements. Converting to GeoJSON...\n", len(osmResp.Elements))

	fc := geojson.NewFeatureCollection()
	for _, element := range osmResp.Elements {
		if element.Type == "relation" {
			var coords [][]float64
			for _, member := range element.Members {
				if member.Role == "outer" && len(member.Geometry) > 0 {
					for _, point := range member.Geometry {
						coords = append(coords, []float64{point.Lon, point.Lat})
					}
					break // take first outer
				}
			}
			if len(coords) > 0 {
				// Close the polygon if not closed
				if coords[0][0] != coords[len(coords)-1][0] || coords[0][1] != coords[len(coords)-1][1] {
					coords = append(coords, coords[0])
				}
				polygon := geojson.NewPolygonGeometry([][][]float64{coords})
				feature := geojson.NewFeature(polygon)
				for k, v := range element.Tags {
					feature.SetProperty(k, v)
				}
				fc.AddFeature(feature)
			}
		}
	}

	geojsonData, err := json.Marshal(fc)
	if err != nil {
		return err
	}

	outputDir := fmt.Sprintf("geojson/%s/%d", countryCode, adminLevel)
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	outputFile := fmt.Sprintf("%s/boundary.geojson", outputDir)
	err = os.WriteFile(outputFile, geojsonData, 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully exported %d features to %s\n", len(fc.Features), outputFile)
	return nil
}
