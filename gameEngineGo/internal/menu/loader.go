package menu

import (
	"log"

	"school-days-engine/internal/graphics"
)

// loadMenu loads a menu layout and regions (matches C++ load_menu)
func (m *Manager) loadMenu(name string) {
	log.Printf("Loading menu: %s", name)

	// Try to load from .glmap files first
	if err := m.loadMenuFromGLMap(name); err != nil {
		log.Printf("Failed to load glmap for %s: %v", name, err)

		// Fall back to sample regions
		m.clearRegions()
		m.graphics.LoadTexture(name+".png", graphics.LayerMenu)
		m.graphics.LoadTexture(name+"_chip.png", graphics.LayerMenuOverlay)
		m.createSampleRegions(name)
	}
}

// loadMenuFromGLMap loads a menu and its regions from .glmap files (matches C++ load_glmap/load_chip)
func (m *Manager) loadMenuFromGLMap(name string) error {
	log.Printf("Loading menu with glmap: %s", name)

	// Clear existing regions and chips
	m.clearRegions()

	// Load menu graphics
	m.graphics.LoadTexture(name+".png", graphics.LayerMenu)
	m.graphics.LoadTexture(name+"_chip.png", graphics.LayerMenuOverlay)

	// Try to load .glmap file for regions
	glmapPath := "System/" + name + ".glmap"
	if reader, err := m.filesystem.Open(glmapPath); err == nil {
		defer reader.Close()

		if data, err := parseGLMap(reader, false); err == nil {
			for _, glmapRegion := range data.Regions {
				region := &Region{
					Index: glmapRegion.Index,
					X1:    glmapRegion.X1,
					Y1:    glmapRegion.Y1,
					X2:    glmapRegion.X2,
					Y2:    glmapRegion.Y2,
					State: glmapRegion.State,
				}
				m.regions = append(m.regions, region)
			}
			log.Printf("Loaded %d regions from %s", len(data.Regions), glmapPath)
		} else {
			log.Printf("Error parsing glmap file %s: %v", glmapPath, err)
		}
	} else {
		log.Printf("Could not load glmap file %s: %v", glmapPath, err)
		// Fall back to sample regions
		m.createSampleRegions(name)
	}

	// Try to load _chip.glmap file for visual feedback
	chipGlmapPath := "System/" + name + "_chip.glmap"
	if reader, err := m.filesystem.Open(chipGlmapPath); err == nil {
		defer reader.Close()

		if data, err := parseGLMap(reader, true); err == nil {
			for _, chipData := range data.Chips {
				chip := &ChipRegion{
					RegionIndex: chipData.RegionIndex,
					State:       chipData.State,
					X1:          chipData.X1,
					Y1:          chipData.Y1,
					X2:          chipData.X2,
					Y2:          chipData.Y2,
				}
				m.chips = append(m.chips, chip)
			}
			log.Printf("Loaded %d chips from %s", len(data.Chips), chipGlmapPath)
		} else {
			log.Printf("Error parsing chip glmap file %s: %v", chipGlmapPath, err)
		}
	} else {
		log.Printf("Could not load chip glmap file %s: %v", chipGlmapPath, err)
	}

	return nil
}
