package graphics

import (
	"image"
	"image/color"
	_ "image/jpeg" // JPEG decoder
	_ "image/png"  // PNG decoder
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Layer constants based on the original C++ engine
const (
	LayerBG          = 0
	LayerBGOverlay0  = 1
	LayerBGOverlay1  = 2
	LayerBGOverlay2  = 3
	LayerTitleBase   = 4
	LayerMenu        = 5
	LayerMenuOverlay = 6
	LayerSysBase     = 7
	LayerDlg         = 8
	LayerDlgOverlay  = 9
	LayerOverlay     = 10
	LayersCount      = 11
)

// LayerState represents the state of a layer
type LayerState struct {
	Visible bool
	Alpha   float64
	X, Y    int
	ScaleX  float64
	ScaleY  float64
	SrcRect image.Rectangle
	DstRect image.Rectangle
}

// Renderer handles all 2D graphics rendering using Ebiten
type Renderer struct {
	screenWidth  int
	screenHeight int

	// Layer textures
	layers      [LayersCount]*ebiten.Image
	layerStates [LayersCount]LayerState

	// Fade effect
	fadeTexture *ebiten.Image
	fadeAlpha   float64
	fadeToWhite bool

	// Special textures
	blackTexture *ebiten.Image
	whiteTexture *ebiten.Image
}

// NewRenderer creates a new graphics renderer
func NewRenderer(width, height int) *Renderer {
	return &Renderer{
		screenWidth:  width,
		screenHeight: height,
	}
}

// Init initializes the graphics renderer
func (r *Renderer) Init() error {
	// Initialize layer states
	for i := range r.layerStates {
		r.layerStates[i] = LayerState{
			Visible: false,
			Alpha:   1.0,
			X:       0,
			Y:       0,
			ScaleX:  1.0,
			ScaleY:  1.0,
			SrcRect: image.Rect(0, 0, r.screenWidth, r.screenHeight),
			DstRect: image.Rect(0, 0, r.screenWidth, r.screenHeight),
		}
	}

	// Create black and white textures for fade effects
	r.blackTexture = ebiten.NewImage(r.screenWidth, r.screenHeight)
	r.blackTexture.Fill(color.RGBA{0, 0, 0, 255})

	r.whiteTexture = ebiten.NewImage(r.screenWidth, r.screenHeight)
	r.whiteTexture.Fill(color.RGBA{255, 255, 255, 255})

	log.Println("Graphics renderer initialized")
	return nil
}

// LoadTexture loads a texture into the specified layer
func (r *Renderer) LoadTexture(filename string, layer int) error {
	if layer < 0 || layer >= LayersCount {
		return nil
	}

	// Try to load the actual image file first
	fullPath := filepath.Join("assets", filename)

	// Check if file exists
	if _, err := os.Stat(fullPath); err == nil {
		// File exists, try to load it
		img, _, err := ebitenutil.NewImageFromFile(fullPath)
		if err != nil {
			log.Printf("Failed to load texture %s: %v, using placeholder", filename, err)
		} else {
			r.layers[layer] = img
			r.layerStates[layer].Visible = true
			log.Printf("Loaded texture %s into layer %d", filename, layer)
			return nil
		}
	}

	// Fall back to placeholder if file doesn't exist or failed to load
	img := ebiten.NewImage(r.screenWidth, r.screenHeight)

	// Create different colors for different layers for testing
	var fillColor color.RGBA
	switch layer {
	case LayerBG:
		fillColor = color.RGBA{50, 50, 100, 255} // Dark blue background
	case LayerTitleBase:
		fillColor = color.RGBA{100, 50, 50, 255} // Dark red
	case LayerMenu:
		fillColor = color.RGBA{50, 100, 50, 255} // Dark green
	default:
		fillColor = color.RGBA{80, 80, 80, 255} // Gray
	}

	img.Fill(fillColor)

	r.layers[layer] = img
	r.layerStates[layer].Visible = true

	log.Printf("Created placeholder texture for %s in layer %d", filename, layer)
	return nil
}

// UnloadTexture removes a texture from the specified layer
func (r *Renderer) UnloadTexture(layer int) {
	if layer < 0 || layer >= LayersCount {
		return
	}

	r.layers[layer] = nil
	r.layerStates[layer].Visible = false

	log.Printf("Unloaded texture from layer %d", layer)
}

// SetLayerVisible sets the visibility of a layer
func (r *Renderer) SetLayerVisible(layer int, visible bool) {
	if layer < 0 || layer >= LayersCount {
		return
	}
	r.layerStates[layer].Visible = visible
}

// SetLayerAlpha sets the alpha transparency of a layer
func (r *Renderer) SetLayerAlpha(layer int, alpha float64) {
	if layer < 0 || layer >= LayersCount {
		return
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	r.layerStates[layer].Alpha = alpha
}

// SetLayerPosition sets the position of a layer
func (r *Renderer) SetLayerPosition(layer int, x, y int) {
	if layer < 0 || layer >= LayersCount {
		return
	}
	r.layerStates[layer].X = x
	r.layerStates[layer].Y = y

	// Update destination rectangle
	state := &r.layerStates[layer]
	width := state.DstRect.Dx()
	height := state.DstRect.Dy()
	state.DstRect = image.Rect(x, y, x+width, y+height)
}

// SetLayerScale sets the scale of a layer
func (r *Renderer) SetLayerScale(layer int, scaleX, scaleY float64) {
	if layer < 0 || layer >= LayersCount {
		return
	}
	r.layerStates[layer].ScaleX = scaleX
	r.layerStates[layer].ScaleY = scaleY
}

// SetFade sets the fade effect
func (r *Renderer) SetFade(alpha float64, toWhite bool) {
	r.fadeAlpha = alpha
	r.fadeToWhite = toWhite

	if alpha > 0 {
		if toWhite {
			r.fadeTexture = r.whiteTexture
		} else {
			r.fadeTexture = r.blackTexture
		}
	} else {
		r.fadeTexture = nil
	}
}

// Draw renders all layers to the screen
func (r *Renderer) Draw(screen *ebiten.Image) {
	// Draw layers in order from back to front
	for i := 0; i < LayersCount; i++ {
		if r.layers[i] != nil && r.layerStates[i].Visible {
			r.drawLayer(screen, i)
		}
	}

	// Draw fade overlay if active
	if r.fadeTexture != nil && r.fadeAlpha > 0 {
		opts := &ebiten.DrawImageOptions{}
		opts.ColorScale.ScaleAlpha(float32(r.fadeAlpha))
		screen.DrawImage(r.fadeTexture, opts)
	}
}

// drawLayer draws a single layer
func (r *Renderer) drawLayer(screen *ebiten.Image, layer int) {
	state := &r.layerStates[layer]
	opts := &ebiten.DrawImageOptions{}

	// Apply alpha
	opts.ColorScale.ScaleAlpha(float32(state.Alpha))

	// Apply scaling
	if state.ScaleX != 1.0 || state.ScaleY != 1.0 {
		opts.GeoM.Scale(state.ScaleX, state.ScaleY)
	}

	// Apply translation
	opts.GeoM.Translate(float64(state.X), float64(state.Y))

	// Draw the layer
	screen.DrawImage(r.layers[layer], opts)
}

// GetScreenSize returns the screen dimensions
func (r *Renderer) GetScreenSize() (int, int) {
	return r.screenWidth, r.screenHeight
}
