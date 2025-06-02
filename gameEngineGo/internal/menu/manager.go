package menu

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"school-days-engine/internal/audio"
	"school-days-engine/internal/graphics"
	"school-days-engine/internal/input"
)

// Menu states based on the original C++ engine
const (
	MenuInit          = 0
	MenuSplash        = 1
	MenuTitle         = 2
	MenuPreTitle      = 102
	MenuExitDlg       = 3
	MenuExit          = 4
	MenuSettings      = 5
	MenuPreSettings   = 105
	MenuSettingsSound = 6
	MenuLoad          = 7
	MenuPreLoad       = 107
	MenuReplay        = 8
	MenuPreReplay     = 108
	MenuRouteMap      = 9
	MenuNewGame       = 10
	MenuGame          = 11
	MenuSave          = 12
	MenuHistory       = 13
	MenuTitleDlg      = 14
	MenuClose         = 15
)

// Region states
const (
	MenuDefault       = 0
	MenuMouseOver     = 1
	MenuDisable       = 2
	MenuSelected      = 3
	MenuSelectedMouse = 4
)

// Region represents an interactive region on screen
type Region struct {
	Index  int
	X1, Y1 float64
	X2, Y2 float64
	State  int
}

// Manager handles menu system and user interface
type Manager struct {
	graphics *graphics.Renderer
	audio    *audio.Manager
	input    *input.Manager

	state     int
	nextState int
	inGame    bool
	dlgActive bool

	regions    []*Region
	dlgRegions [2]*Region

	debugMode bool
}

// NewManager creates a new menu manager
func NewManager(gfx *graphics.Renderer, aud *audio.Manager, inp *input.Manager) *Manager {
	return &Manager{
		graphics:  gfx,
		audio:     aud,
		input:     inp,
		state:     MenuInit,
		inGame:    false,
		dlgActive: false,
		regions:   make([]*Region, 0),
		debugMode: true, // Enable debug mode for development
	}
}

// Init initializes the menu system
func (m *Manager) Init() error {
	log.Println("Menu manager initialized")
	// Start with splash screen
	m.nextMenuState(MenuInit)

	return nil
}

// Update updates the menu system
func (m *Manager) Update() error {
	// Process input
	m.processInput()

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

// processInput handles user input
func (m *Manager) processInput() {
	mouseX, mouseY := m.input.GetMousePosition()

	// Check right click for going back
	if m.input.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		m.prevState(-1)
		return
	}

	// Update region states based on mouse position
	for _, region := range m.regions {
		if m.pointInRegion(float64(mouseX), float64(mouseY), region) {
			if region.State == MenuDefault {
				region.State = MenuMouseOver
				m.audio.PlaySystemSound(audio.SndSystem) // Hover sound
			}

			// Check for click
			if m.input.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				region.State = MenuSelected
				m.audio.PlaySystemSound(audio.SndSelect) // Click sound
				m.onRegionClicked(region.Index)
			}
		} else {
			if region.State == MenuMouseOver {
				region.State = MenuDefault
			}
		}
	}

	// Handle keyboard shortcuts
	if m.input.IsKeyJustPressed(ebiten.KeyEscape) {
		m.prevState(-1)
	}
}

// pointInRegion checks if a point is inside a region
func (m *Manager) pointInRegion(x, y float64, region *Region) bool {
	return x >= region.X1 && x <= region.X2 && y >= region.Y1 && y <= region.Y2
}

// onRegionClicked handles region click events
func (m *Manager) onRegionClicked(regionIndex int) {
	switch m.state {
	case MenuTitle:
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
	case MenuLoad:
		log.Printf("Load slot %d clicked", regionIndex)
	case MenuSettings:
		switch regionIndex {
		case 0: // Sound Settings
			m.changeToState(MenuSettingsSound)
		case 1: // Back
			m.prevState(-1)
		}
	}
}

// nextMenuState transitions to the next menu state
func (m *Manager) nextMenuState(index int) {
	prevState := m.state

	switch m.state {
	case MenuInit:
		m.state = MenuSplash
		m.showSplash()
	case MenuSplash:
		m.state = MenuTitle
		m.showTitle()
	case MenuTitle:
		switch index {
		case 1: // Load
			m.nextState = MenuLoad
			m.state = MenuPreLoad
			m.menuExitEffect()
		case 3: // Settings
			m.nextState = MenuSettings
			m.state = MenuPreSettings
			m.menuExitEffect()
		case 4: // Exit
			m.state = MenuExitDlg
			m.showExitDialog()
		}
	case MenuPreLoad:
		m.state = MenuLoad
		m.showLoad()
	case MenuPreSettings:
		m.state = MenuSettings
		m.showSettings()
	case MenuLoad:
		if index == -1 { // Back
			m.prevState(-1)
		}
	case MenuSettings:
		if index == -1 { // Back
			m.prevState(-1)
		}
	}

	if prevState != m.state {
		log.Printf("Menu state changed: %d -> %d", prevState, m.state)
	}
}

// prevState goes back to the previous menu state
func (m *Manager) prevState(index int) {
	switch m.state {
	case MenuLoad, MenuSettings, MenuSettingsSound:
		m.state = MenuTitle
		m.showTitle()
	case MenuExitDlg:
		m.state = MenuTitle
		m.showTitle()
	}
}

// changeToState immediately changes to a new state
func (m *Manager) changeToState(newState int) {
	log.Printf("Changing menu state from %d to %d", m.state, newState)
	m.state = newState

	switch newState {
	case MenuLoad:
		m.showLoad()
	case MenuSettings:
		m.showSettings()
	case MenuExitDlg:
		m.showExitDialog()
	case MenuSettingsSound:
		log.Println("Showing sound settings")
		// TODO: Implement sound settings menu
	}
}

// updateState updates the current menu state
func (m *Manager) updateState() {
	// Handle state transitions and animations
	switch m.state {
	case MenuSplash:
		// Auto-advance from splash after a delay
		// For now, advance immediately for testing
		go func() {
			// time.Sleep(2 * time.Second)
			// m.nextState(-1)
		}()
	}
}

// showSplash displays the splash screen
func (m *Manager) showSplash() {
	log.Println("Showing splash screen")

	// Clear regions
	m.regions = m.regions[:0]

	// Load splash image
	m.graphics.LoadTexture("System/Logo.png", graphics.LayerMenu)

	// Set up fade effect
	// TODO: Implement fade effects through script engine
}

// showTitle displays the title screen
func (m *Manager) showTitle() {
	log.Println("Showing title screen")

	// Play title music
	m.audio.PlaySystemSound(audio.SndTitle)

	// Load title menu
	m.loadMenu("Title/Title")
}

// showLoad displays the load game screen
func (m *Manager) showLoad() {
	log.Println("Showing load screen")
	m.loadMenu("Load/Load")
}

// showSettings displays the settings screen
func (m *Manager) showSettings() {
	log.Println("Showing settings screen")
	m.loadMenu("Settings/Settings")
}

// showExitDialog displays the exit confirmation dialog
func (m *Manager) showExitDialog() {
	log.Println("Showing exit dialog")
	m.dlgActive = true
	// TODO: Load dialog graphics and regions
}

// loadMenu loads a menu layout and regions
func (m *Manager) loadMenu(name string) {
	log.Printf("Loading menu: %s", name)

	// Clear existing regions
	m.regions = m.regions[:0]

	// Load menu graphics
	m.graphics.LoadTexture(name+".png", graphics.LayerMenu)
	m.graphics.LoadTexture(name+"_chip.png", graphics.LayerMenuOverlay)

	// Create sample regions for testing
	m.createSampleRegions(name)
}

// createSampleRegions creates sample interactive regions for testing
func (m *Manager) createSampleRegions(menuName string) {
	switch menuName {
	case "Title/Title":
		// Create regions for title menu buttons
		m.regions = append(m.regions, &Region{0, 100, 200, 300, 250, MenuDefault}) // New Game
		m.regions = append(m.regions, &Region{1, 100, 270, 300, 320, MenuDefault}) // Load
		m.regions = append(m.regions, &Region{2, 100, 340, 300, 390, MenuDefault}) // Replay
		m.regions = append(m.regions, &Region{3, 100, 410, 300, 460, MenuDefault}) // Settings
		m.regions = append(m.regions, &Region{4, 100, 480, 300, 530, MenuDefault}) // Exit

	case "Load/Load":
		// Create regions for save slots
		for i := 0; i < 5; i++ {
			y1 := float64(100 + i*80)
			y2 := y1 + 60
			m.regions = append(m.regions, &Region{i, 50, y1, 750, y2, MenuDefault})
		}

	case "Settings/Settings":
		// Create regions for settings options
		m.regions = append(m.regions, &Region{0, 100, 200, 400, 250, MenuDefault}) // Sound
		m.regions = append(m.regions, &Region{1, 100, 500, 200, 550, MenuDefault}) // Back
	}

	log.Printf("Created %d regions for menu %s", len(m.regions), menuName)
}

// menuExitEffect plays an exit transition effect
func (m *Manager) menuExitEffect() {
	log.Println("Playing menu exit effect")
	// TODO: Implement fade out effect
}

// GetState returns the current menu state
func (m *Manager) GetState() int {
	return m.state
}

// InDialog returns whether a dialog is active
func (m *Manager) InDialog() bool {
	return m.dlgActive
}

// GetRegions returns the current interactive regions
func (m *Manager) GetRegions() []*Region {
	return m.regions
}
