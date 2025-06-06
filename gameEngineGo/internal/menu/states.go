package menu

import "log"

// Menu states based on the original C++ engine
const (
	MenuInit = iota
	MenuSplash
	MenuTitle
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

// Region states using iota for better Go practices
const (
	MenuDefault = iota
	MenuMouseOver
	MenuDisable
	MenuSelected
	MenuSelectedMouse
)

// StateInfo provides metadata about menu states
type StateInfo struct {
	Name        string
	Description string
}

// GetStateInfo returns information about a menu state
func GetStateInfo(state int) StateInfo {
	switch state {
	case MenuInit:
		return StateInfo{"INIT", "Initialization"}
	case MenuSplash:
		return StateInfo{"SPLASH", "Splash screen"}
	case MenuTitle:
		return StateInfo{"TITLE", "Title screen"}
	case MenuLoad:
		return StateInfo{"LOAD", "Load game"}
	case MenuSettings:
		return StateInfo{"SETTINGS", "Settings menu"}
	case MenuExitDlg:
		return StateInfo{"EXIT_DLG", "Exit confirmation"}
	default:
		return StateInfo{"UNKNOWN", "Unknown state"}
	}
}

// nextMenuState transitions to the next menu state (matches C++ behavior)
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
			m.prevState()
		}
	case MenuSettings:
		if index == -1 { // Back
			m.prevState()
		}
	}

	if prevState != m.state {
		log.Printf("Menu state changed: %d -> %d", prevState, m.state)
	}
}

// prevState goes back to the previous menu state
func (m *Manager) prevState() {
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

// updateState updates the current menu state machine
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
	m.clearRegions()
	m.graphics.LoadTexture("System/Logo.png", 0) // LayerMenu
}

// showTitle displays the title screen
func (m *Manager) showTitle() {
	log.Println("Showing title screen")
	m.audio.PlaySystemSoundByID(1) // SndTitle
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

// menuExitEffect plays an exit transition effect
func (m *Manager) menuExitEffect() {
	log.Println("Playing menu exit effect")
	// TODO: Implement fade out effect
}
