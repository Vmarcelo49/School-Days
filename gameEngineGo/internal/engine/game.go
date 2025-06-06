package engine

import (
	"fmt"
	"image/color"
	"log"

	"school-days-engine/internal/audio"
	"school-days-engine/internal/filesystem"
	"school-days-engine/internal/graphics"
	"school-days-engine/internal/input"
	"school-days-engine/internal/menu"
	"school-days-engine/internal/script"
	"school-days-engine/internal/settings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Game represents the main game engine
type Game struct {
	graphics   *graphics.Renderer
	audio      *audio.Manager
	input      *input.Manager
	filesystem *filesystem.Manager
	script     *script.Engine
	menu       *menu.Manager
	settings   *settings.Manager

	screenWidth  int
	screenHeight int

	initialized bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		screenWidth:  800,
		screenHeight: 600,
	}
}

// Init initializes all game subsystems
func (g *Game) Init() error {
	var err error

	// Initialize settings
	g.settings = settings.NewManager("./config/settings.json")
	if err = g.settings.Load(); err != nil {
		log.Printf("Warning: failed to load settings: %v", err)
	}

	// Use settings for screen size
	config := g.settings.GetConfig()
	g.screenWidth = config.ScreenWidth
	g.screenHeight = config.ScreenHeight

	// Initialize filesystem
	g.filesystem = filesystem.NewManager("./")
	if err = g.filesystem.Init(); err != nil {
		return fmt.Errorf("failed to initialize filesystem: %w", err)
	}
	// Initialize graphics renderer
	g.graphics = graphics.NewRenderer(g.screenWidth, g.screenHeight, g.filesystem)
	if err = g.graphics.Init(); err != nil {
		return fmt.Errorf("failed to initialize graphics: %w", err)
	}

	// Initialize audio manager
	g.audio = audio.NewManager()
	if err = g.audio.Init(); err != nil {
		return fmt.Errorf("failed to initialize audio: %w", err)
	}

	// Initialize input manager
	g.input = input.NewManager()

	// Initialize script engine
	g.script = script.NewEngine()
	if err = g.script.Init(); err != nil {
		return fmt.Errorf("failed to initialize script engine: %w", err)
	}
	// Initialize menu system
	g.menu = menu.NewManager(g.graphics, g.audio, g.input, g.filesystem, g.screenWidth, g.screenHeight)
	if err = g.menu.Init(); err != nil {
		return fmt.Errorf("failed to initialize menu: %w", err)
	}

	g.initialized = true
	log.Println("Game engine initialized successfully")
	return nil
}

// Update updates the game logic
func (g *Game) Update() error {
	if !g.initialized {
		return nil
	}

	// Update input
	g.input.Update()

	// Update script engine
	if err := g.script.Update(); err != nil {
		return err
	}

	// Update menu system
	if err := g.menu.Update(); err != nil {
		return err
	}

	// Update audio
	g.audio.Update()

	return nil
}

// Draw renders the game
func (g *Game) Draw(screen *ebiten.Image) {
	if !g.initialized {
		screen.Fill(color.RGBA{0, 0, 0, 255})
		ebitenutil.DebugPrint(screen, "Initializing...")
		return
	}

	// Clear screen
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw all layers through graphics renderer
	g.graphics.Draw(screen)

	// Draw menu system
	g.menu.Draw(screen)

	// Debug info
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.2f", ebiten.ActualFPS()))
}

// Layout returns the game's screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.screenWidth, g.screenHeight
}

// Run starts the game
func (g *Game) Run() error {
	// Initialize the game
	if err := g.Init(); err != nil {
		return err
	}

	// Set window properties
	ebiten.SetWindowSize(g.screenWidth, g.screenHeight)
	ebiten.SetWindowTitle("School Days Engine - Go Port")
	ebiten.SetWindowResizable(false)

	// Run the game loop
	if err := ebiten.RunGame(g); err != nil {
		return err
	}

	return nil
}
