package menu

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"school-days-engine/internal/audio"
	"school-days-engine/internal/filesystem"
	"school-days-engine/internal/graphics"
	"school-days-engine/internal/input"
)

// Manager handles menu system and user interface (matches C++ Menu class)
type Manager struct {
	graphics   *graphics.Renderer
	audio      *audio.Manager
	input      *input.Manager
	filesystem *filesystem.Manager

	state     int
	nextState int
	inGame    bool
	dlgActive bool

	regions    []*Region
	chips      []*ChipRegion
	dlgRegions [2]*Region
	dlgChips   [2]*ChipRegion

	screenWidth  int
	screenHeight int
	debugMode    bool
}

// NewManager creates a new menu manager (matches C++ Menu constructor)
func NewManager(gfx *graphics.Renderer, aud *audio.Manager, inp *input.Manager, fs *filesystem.Manager, screenW, screenH int) *Manager {
	return &Manager{
		graphics:     gfx,
		audio:        aud,
		input:        inp,
		filesystem:   fs,
		state:        MenuInit,
		inGame:       false,
		dlgActive:    false,
		regions:      make([]*Region, 0),
		chips:        make([]*ChipRegion, 0),
		screenWidth:  screenW,
		screenHeight: screenH,
		debugMode:    true, // Enable debug mode for development
	}
}

// Init initializes the menu system (matches C++ Menu initialization)
func (m *Manager) Init() error {
	log.Println("Menu manager initialized")
	// Start with splash screen
	m.nextMenuState(MenuInit)
	return nil
}

// Update updates the menu system (matches C++ Menu::proc)
func (m *Manager) Update() error {
	// Process input
	m.processInput()

	// Update region chip associations
	m.updateRegionChips()

	// Update menu state machine
	m.updateState()

	return nil
}

// Draw renders the menu system
func (m *Manager) Draw(screen *ebiten.Image) {
	// Draw debug information
	if m.debugMode {
		debugText := "Menu State: "
		switch m.state {
		case MenuInit:
			debugText += "INIT"
		case MenuSplash:
			debugText += "SPLASH"
		case MenuTitle:
			debugText += "TITLE"
		case MenuLoad:
			debugText += "LOAD"
		case MenuSettings:
			debugText += "SETTINGS"
		default:
			debugText += "UNKNOWN"
		}

		ebitenutil.DebugPrintAt(screen, debugText, 10, 30)

		// Draw regions for debugging
		mouseX, mouseY := m.input.GetMousePosition()
		ebitenutil.DebugPrintAt(screen, "Mouse: "+fmt.Sprintf("%d, %d", mouseX, mouseY), 10, 50)

		// Draw interactive regions
		for i, region := range m.regions {
			regionText := fmt.Sprintf("Region %d: (%.0f,%.0f)-(%.0f,%.0f) State:%d",
				i, region.X1, region.Y1, region.X2, region.Y2, region.State)
			ebitenutil.DebugPrintAt(screen, regionText, 10, 70+i*20)
		}
	}
}

// GetState returns the current menu state
func (m *Manager) GetState() int {
	return m.state
}

// InDialog returns whether a dialog is active
func (m *Manager) InDialog() bool {
	return m.dlgActive
}
