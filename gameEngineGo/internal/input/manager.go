package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MouseState represents the current mouse state
type MouseState struct {
	X, Y         int
	LeftButton   bool
	RightButton  bool
	LeftPressed  bool
	RightPressed bool
}

// Manager handles all input from keyboard and mouse
type Manager struct {
	mouse     MouseState
	prevMouse MouseState
	keys      map[ebiten.Key]bool
	prevKeys  map[ebiten.Key]bool
}

// NewManager creates a new input manager
func NewManager() *Manager {
	return &Manager{
		keys:     make(map[ebiten.Key]bool),
		prevKeys: make(map[ebiten.Key]bool),
	}
}

// Update updates the input state
func (m *Manager) Update() {
	// Store previous mouse state
	m.prevMouse = m.mouse

	// Update mouse position
	m.mouse.X, m.mouse.Y = ebiten.CursorPosition()

	// Update mouse buttons
	m.mouse.LeftButton = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	m.mouse.RightButton = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)

	// Check for mouse button presses (just pressed this frame)
	m.mouse.LeftPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	m.mouse.RightPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)

	// Store previous key states
	for k, v := range m.keys {
		m.prevKeys[k] = v
	}

	// Update key states
	m.keys[ebiten.KeyEscape] = ebiten.IsKeyPressed(ebiten.KeyEscape)
	m.keys[ebiten.KeyEnter] = ebiten.IsKeyPressed(ebiten.KeyEnter)
	m.keys[ebiten.KeySpace] = ebiten.IsKeyPressed(ebiten.KeySpace)
	m.keys[ebiten.KeyArrowUp] = ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	m.keys[ebiten.KeyArrowDown] = ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	m.keys[ebiten.KeyArrowLeft] = ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	m.keys[ebiten.KeyArrowRight] = ebiten.IsKeyPressed(ebiten.KeyArrowRight)
}

// GetMousePosition returns the current mouse position
func (m *Manager) GetMousePosition() (int, int) {
	return m.mouse.X, m.mouse.Y
}

// IsMouseButtonPressed returns true if the specified mouse button is currently pressed
func (m *Manager) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	switch button {
	case ebiten.MouseButtonLeft:
		return m.mouse.LeftButton
	case ebiten.MouseButtonRight:
		return m.mouse.RightButton
	default:
		return false
	}
}

// IsMouseButtonJustPressed returns true if the specified mouse button was just pressed this frame
func (m *Manager) IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	switch button {
	case ebiten.MouseButtonLeft:
		return m.mouse.LeftPressed
	case ebiten.MouseButtonRight:
		return m.mouse.RightPressed
	default:
		return false
	}
}

// IsKeyPressed returns true if the specified key is currently pressed
func (m *Manager) IsKeyPressed(key ebiten.Key) bool {
	return m.keys[key]
}

// IsKeyJustPressed returns true if the specified key was just pressed this frame
func (m *Manager) IsKeyJustPressed(key ebiten.Key) bool {
	return m.keys[key] && !m.prevKeys[key]
}

// GetMouseState returns the current mouse state
func (m *Manager) GetMouseState() MouseState {
	return m.mouse
}

// GetNormalizedMousePosition returns mouse position normalized to 0.0-1.0 range
func (m *Manager) GetNormalizedMousePosition(screenWidth, screenHeight int) (float64, float64) {
	if screenWidth <= 0 || screenHeight <= 0 {
		return 0.0, 0.0
	}

	normalizedX := float64(m.mouse.X) / float64(screenWidth)
	normalizedY := float64(m.mouse.Y) / float64(screenHeight)

	// Clamp to valid range
	if normalizedX < 0.0 {
		normalizedX = 0.0
	} else if normalizedX > 1.0 {
		normalizedX = 1.0
	}

	if normalizedY < 0.0 {
		normalizedY = 0.0
	} else if normalizedY > 1.0 {
		normalizedY = 1.0
	}

	return normalizedX, normalizedY
}

// CheckPointInRegion checks if normalized coordinates are within a normalized region
func (m *Manager) CheckPointInRegion(normX, normY, x1, y1, x2, y2 float64) bool {
	return normX >= x1 && normX <= x2 && normY >= y1 && normY <= y2
}
