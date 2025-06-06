package menu

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"school-days-engine/internal/audio"
)

// Region represents an interactive region on screen with normalized coordinates
type Region struct {
	Index  int
	X1, Y1 float64 // Normalized coordinates (0.0-1.0)
	X2, Y2 float64 // Normalized coordinates (0.0-1.0)
	State  int
	Chip   *ChipRegion // Visual feedback chip
}

// ChipRegion represents visual feedback for region states
type ChipRegion struct {
	RegionIndex int
	State       int
	X1, Y1      float64 // Texture coordinates for chip
	X2, Y2      float64 // Texture coordinates for chip
}

// IsMouseOver checks if the region contains the given normalized coordinates
func (r *Region) IsMouseOver(normX, normY float64) bool {
	return normX >= r.X1 && normX <= r.X2 && normY >= r.Y1 && normY <= r.Y2
}

// SetState safely updates the region state with validation
func (r *Region) SetState(newState int) {
	if newState >= MenuDefault && newState <= MenuSelectedMouse {
		r.State = newState
	}
}

// String provides a readable representation of the region
func (r *Region) String() string {
	return fmt.Sprintf("Region %d: (%.3f,%.3f)-(%.3f,%.3f) State:%d",
		r.Index, r.X1, r.Y1, r.X2, r.Y2, r.State)
}

// clearRegions clears all interactive regions
func (m *Manager) clearRegions() {
	m.regions = m.regions[:0]
	m.chips = m.chips[:0]
}

// processInput handles user input and region interaction (matches C++ region_check)
func (m *Manager) processInput() {
	// Get normalized mouse coordinates
	normX, normY := m.input.GetNormalizedMousePosition(m.screenWidth, m.screenHeight)

	// Check right click for going back
	if m.input.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		m.prevState()
		return
	}

	// Handle dialog input separately if active
	if m.dlgActive {
		m.processDialogInput(normX, normY)
		return
	}

	// Update region states based on mouse position using normalized coordinates
	for _, region := range m.regions {
		if region.State == MenuDisable {
			continue // Skip disabled regions
		}

		if region.IsMouseOver(normX, normY) {
			// Mouse is over region
			m.handleRegionMouseOver(region)

			// Check for click
			if m.input.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				m.handleRegionClick(region)
			}
		} else {
			// Mouse is not over region - revert hover states
			m.handleRegionMouseLeave(region)
		}
	}

	// Handle keyboard shortcuts
	if m.input.IsKeyJustPressed(ebiten.KeyEscape) {
		m.prevState()
	}
}

// processDialogInput handles input when a dialog is active
func (m *Manager) processDialogInput(normX, normY float64) {
	// Process dialog regions similar to main regions
	for i := range m.dlgRegions {
		region := m.dlgRegions[i]
		if region == nil {
			continue
		}

		if region.IsMouseOver(normX, normY) {
			m.handleRegionMouseOver(region)
			if m.input.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				m.handleDialogRegionClick(i)
			}
		} else {
			m.handleRegionMouseLeave(region)
		}
	}
}

// handleRegionMouseOver handles mouse entering a region
func (m *Manager) handleRegionMouseOver(region *Region) {
	if region.State == MenuDefault {
		region.SetState(MenuMouseOver)
		m.audio.PlaySystemSoundByID(audio.SndSystem) // Hover sound
	} else if region.State == MenuSelected {
		region.SetState(MenuSelectedMouse)
	}
}

// handleRegionMouseLeave handles mouse leaving a region
func (m *Manager) handleRegionMouseLeave(region *Region) {
	if region.State == MenuMouseOver {
		region.SetState(MenuDefault)
	} else if region.State == MenuSelectedMouse {
		region.SetState(MenuSelected)
	}
}

// handleRegionClick handles region click events
func (m *Manager) handleRegionClick(region *Region) {
	region.SetState(MenuSelected)
	m.audio.PlaySystemSoundByID(audio.SndSelect) // Click sound
	m.onRegionClicked(region.Index)
}

// handleDialogRegionClick handles dialog region click events
func (m *Manager) handleDialogRegionClick(regionIndex int) {
	switch regionIndex {
	case 0: // Yes/OK
		m.dlgActive = false
		if m.state == MenuExitDlg {
			m.changeToState(MenuExit)
		}
	case 1: // No/Cancel
		m.dlgActive = false
		m.prevState()
	}
}

// onRegionClicked handles region click events (matches C++ next_state)
func (m *Manager) onRegionClicked(regionIndex int) {
	switch m.state {
	case MenuTitle:
		m.handleTitleMenuClick(regionIndex)
	case MenuLoad:
		m.handleLoadMenuClick(regionIndex)
	case MenuSettings:
		m.handleSettingsMenuClick(regionIndex)
	}
}

// handleTitleMenuClick handles title menu button clicks
func (m *Manager) handleTitleMenuClick(regionIndex int) {
	switch regionIndex {
	case 0: // New Game
		log.Println("New Game clicked - not implemented")
	case 1: // Load Game
		m.changeToState(MenuLoad)
	case 2: // Replay
		log.Println("Replay clicked - not implemented")
	case 3: // Settings
		m.changeToState(MenuSettings)
	case 4: // Exit
		m.changeToState(MenuExitDlg)
	}
}

// handleLoadMenuClick handles load menu interactions
func (m *Manager) handleLoadMenuClick(regionIndex int) {
	log.Printf("Load slot %d clicked", regionIndex)
	// TODO: Implement save file loading
}

// handleSettingsMenuClick handles settings menu interactions
func (m *Manager) handleSettingsMenuClick(regionIndex int) {
	switch regionIndex {
	case 0: // Sound Settings
		m.changeToState(MenuSettingsSound)
	case 1: // Back
		m.prevState()
	}
}

// updateRegionChips updates region chip associations based on current states (matches C++ chip assignment)
func (m *Manager) updateRegionChips() {
	// Clear existing chip assignments
	for _, region := range m.regions {
		region.Chip = nil
	}

	// Assign chips to regions based on matching states (matches C++ logic)
	for _, chip := range m.chips {
		if chip.RegionIndex-1 >= 0 && chip.RegionIndex-1 < len(m.regions) {
			region := m.regions[chip.RegionIndex-1]
			if region.State == chip.State {
				region.Chip = chip
			}
		}
	}
}

// createSampleRegions creates sample interactive regions for testing (fallback when glmap fails)
func (m *Manager) createSampleRegions(menuName string) {
	switch menuName {
	case "Title/Title":
		// Create regions for title menu buttons using normalized coordinates
		m.regions = append(m.regions, &Region{Index: 0, X1: 0.125, Y1: 0.625, X2: 0.375, Y2: 0.781, State: MenuDefault}) // New Game
		m.regions = append(m.regions, &Region{Index: 1, X1: 0.125, Y1: 0.844, X2: 0.375, Y2: 1.000, State: MenuDefault}) // Load
		m.regions = append(m.regions, &Region{Index: 2, X1: 0.125, Y1: 1.063, X2: 0.375, Y2: 1.219, State: MenuDefault}) // Replay
		m.regions = append(m.regions, &Region{Index: 3, X1: 0.125, Y1: 1.281, X2: 0.375, Y2: 1.438, State: MenuDefault}) // Settings
		m.regions = append(m.regions, &Region{Index: 4, X1: 0.125, Y1: 1.500, X2: 0.375, Y2: 1.656, State: MenuDefault}) // Exit

	case "Load/Load":
		// Create regions for save slots using normalized coordinates
		for i := range 5 {
			y1 := 0.156 + float64(i)*0.125 // Evenly spaced slots
			y2 := y1 + 0.094               // Height of each slot
			m.regions = append(m.regions, &Region{Index: i, X1: 0.063, Y1: y1, X2: 0.938, Y2: y2, State: MenuDefault})
		}

	case "Settings/Settings":
		// Create regions for settings options using normalized coordinates
		m.regions = append(m.regions, &Region{Index: 0, X1: 0.125, Y1: 0.625, X2: 0.500, Y2: 0.781, State: MenuDefault}) // Sound
		m.regions = append(m.regions, &Region{Index: 1, X1: 0.125, Y1: 1.563, X2: 0.250, Y2: 1.719, State: MenuDefault}) // Back
	}

	log.Printf("Created %d regions for menu %s", len(m.regions), menuName)
}

// GetRegions returns the current interactive regions
func (m *Manager) GetRegions() []*Region {
	return m.regions
}

// debugDrawRegions draws region debug information
func (m *Manager) debugDrawRegions(screen *ebiten.Image) {
	if !m.debugMode {
		return
	}

	// This method is called from Draw() in manager.go
	// Debug info is currently handled there to avoid duplication
}
