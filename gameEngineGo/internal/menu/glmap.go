package menu

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GLMapRegion represents a region loaded from a .glmap file
type GLMapRegion struct {
	Index  int     // Region index (1-based in file, 0-based internally)
	State  int     // Initial state
	X1, Y1 float64 // Top-left coordinates (normalized 0.0-1.0)
	X2, Y2 float64 // Bottom-right coordinates (normalized 0.0-1.0)
}

// GLMapChipRegion represents a chip region for visual state feedback
type GLMapChipRegion struct {
	RegionIndex int     // Which region this chip belongs to
	State       int     // State this chip represents
	X1, Y1      float64 // Texture coordinates (normalized 0.0-1.0)
	X2, Y2      float64 // Texture coordinates (normalized 0.0-1.0)
}

// GLMapData holds all parsed data from .glmap files
type GLMapData struct {
	Regions []GLMapRegion
	Chips   []GLMapChipRegion
}

// ParseGLMapFromReader parses a .glmap file from a reader
func ParseGLMapFromReader(reader io.Reader, isChip bool) (*GLMapData, error) {
	return parseGLMap(reader, isChip)
}

// parseGLMap parses a .glmap file from a reader
func parseGLMap(reader io.Reader, isChip bool) (*GLMapData, error) {
	data := &GLMapData{
		Regions: make([]GLMapRegion, 0),
		Chips:   make([]GLMapChipRegion, 0),
	}

	// First pass: read all lines and build a map of regions
	lines := make(map[string]string)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		if strings.Contains(line, "]=") {
			parts := strings.SplitN(line, "]=", 2)
			if len(parts) == 2 {
				key := strings.Trim(parts[0], "[]")
				lines[key] = parts[1]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading glmap file: %v", err)
	}

	// Get region count (matches C++ behavior)
	regionCountStr, exists := lines["Regions"]
	if !exists {
		return nil, fmt.Errorf("missing [Regions] declaration")
	}

	regionCount, err := strconv.Atoi(regionCountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid region count: %v", err)
	}

	// Process regions in order (1-based indexing like C++)
	for i := 1; i <= regionCount; i++ {
		regionKey := fmt.Sprintf("Region%d", i)
		regionData, exists := lines[regionKey]
		if !exists {
			return nil, fmt.Errorf("missing region data for Region%d", i)
		}

		values := strings.Fields(regionData)
		if len(values) < 5 {
			return nil, fmt.Errorf("insufficient region data for Region%d: %s", i, regionData)
		}

		if isChip {
			// Chip format: region_index state x1 y1 x2 y2
			if len(values) < 6 {
				return nil, fmt.Errorf("insufficient chip data for Region%d: %s", i, regionData)
			}

			chipRegionIndex, err := strconv.Atoi(values[0])
			if err != nil {
				return nil, fmt.Errorf("invalid chip region index for Region%d: %v", i, err)
			}

			state, err := strconv.Atoi(values[1])
			if err != nil {
				return nil, fmt.Errorf("invalid chip state for Region%d: %v", i, err)
			}

			x1, err := strconv.ParseFloat(values[2], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid chip x1 for Region%d: %v", i, err)
			}

			y1, err := strconv.ParseFloat(values[3], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid chip y1 for Region%d: %v", i, err)
			}

			x2, err := strconv.ParseFloat(values[4], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid chip x2 for Region%d: %v", i, err)
			}

			y2, err := strconv.ParseFloat(values[5], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid chip y2 for Region%d: %v", i, err)
			}

			chip := GLMapChipRegion{
				RegionIndex: chipRegionIndex,
				State:       state,
				X1:          x1,
				Y1:          y1,
				X2:          x2,
				Y2:          y2,
			}
			data.Chips = append(data.Chips, chip)
		} else {
			// Standard region format: state x1 y1 x2 y2
			state, err := strconv.Atoi(values[0])
			if err != nil {
				return nil, fmt.Errorf("invalid region state for Region%d: %v", i, err)
			}

			x1, err := strconv.ParseFloat(values[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid region x1 for Region%d: %v", i, err)
			}

			y1, err := strconv.ParseFloat(values[2], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid region y1 for Region%d: %v", i, err)
			}

			x2, err := strconv.ParseFloat(values[3], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid region x2 for Region%d: %v", i, err)
			}

			y2, err := strconv.ParseFloat(values[4], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid region y2 for Region%d: %v", i, err)
			}

			region := GLMapRegion{
				Index: i - 1, // Convert to 0-based indexing for internal use
				State: state,
				X1:    x1,
				Y1:    y1,
				X2:    x2,
				Y2:    y2,
			}
			data.Regions = append(data.Regions, region)
		}
	}

	return data, nil
}
