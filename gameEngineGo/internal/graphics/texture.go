package graphics

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // JPEG decoder
	_ "image/png"  // PNG decoder
	"log"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// TextureCache manages texture loading and caching
type TextureCache struct {
	filesystem FileSystemInterface
	cache      map[string]*ebiten.Image
	mutex      sync.RWMutex
}

// NewTextureCache creates a new texture cache
func NewTextureCache(filesystem FileSystemInterface) *TextureCache {
	return &TextureCache{
		filesystem: filesystem,
		cache:      make(map[string]*ebiten.Image),
	}
}

// LoadTexture loads a texture from file or cache
func (tc *TextureCache) LoadTexture(filename string) (*ebiten.Image, error) {
	// Normalize filename - add .png extension if not present
	normalizedName := strings.ToLower(filename)
	if !strings.Contains(normalizedName, ".") {
		normalizedName += ".png"
	}

	// Check cache first
	tc.mutex.RLock()
	if cached, exists := tc.cache[normalizedName]; exists {
		tc.mutex.RUnlock()
		return cached, nil
	}
	tc.mutex.RUnlock()

	// Load and decode texture
	texture, err := tc.loadFromFile(normalizedName)
	if err != nil {
		return nil, fmt.Errorf("failed to load texture %s: %v", normalizedName, err)
	}

	// Cache the texture
	tc.mutex.Lock()
	tc.cache[normalizedName] = texture
	tc.mutex.Unlock()

	log.Printf("Loaded and cached texture: %s", normalizedName)
	return texture, nil
}

// loadFromFile loads texture data from filesystem
func (tc *TextureCache) loadFromFile(filename string) (*ebiten.Image, error) {
	// Try to load from filesystem (GPK or regular file)
	data, err := tc.filesystem.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Decode image data
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format: %s): %v", format, err)
	}

	// Convert to Ebiten image
	ebitenImg := ebiten.NewImageFromImage(img)
	return ebitenImg, nil
}

// GetTexture returns a cached texture if available
func (tc *TextureCache) GetTexture(filename string) *ebiten.Image {
	normalizedName := strings.ToLower(filename)
	if !strings.Contains(normalizedName, ".") {
		normalizedName += ".png"
	}

	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	return tc.cache[normalizedName]
}

// ClearCache removes all cached textures
func (tc *TextureCache) ClearCache() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Dispose of all textures
	for name, texture := range tc.cache {
		if texture != nil {
			texture.Dispose()
		}
		delete(tc.cache, name)
	}

	log.Println("Texture cache cleared")
}

// CacheSize returns the number of cached textures
func (tc *TextureCache) CacheSize() int {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	return len(tc.cache)
}

// CreateSolidTexture creates a solid color texture
func (tc *TextureCache) CreateSolidTexture(width, height int, colorValues [4]uint8) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(color.RGBA{R: colorValues[0], G: colorValues[1], B: colorValues[2], A: colorValues[3]})
	return img
}

// TextureManager provides high-level texture management for the renderer
type TextureManager struct {
	cache        *TextureCache
	screenWidth  int
	screenHeight int

	// Special textures
	blackTexture *ebiten.Image
	whiteTexture *ebiten.Image
}

// NewTextureManager creates a new texture manager
func NewTextureManager(filesystem FileSystemInterface, screenWidth, screenHeight int) *TextureManager {
	cache := NewTextureCache(filesystem)

	tm := &TextureManager{
		cache:        cache,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}
	// Create special textures
	tm.blackTexture = cache.CreateSolidTexture(screenWidth, screenHeight, [4]uint8{0, 0, 0, 255})
	tm.whiteTexture = cache.CreateSolidTexture(screenWidth, screenHeight, [4]uint8{255, 255, 255, 255})

	return tm
}

// LoadTexture loads a texture with error handling and fallback
func (tm *TextureManager) LoadTexture(filename string) (*ebiten.Image, error) {
	texture, err := tm.cache.LoadTexture(filename)
	if err != nil {
		log.Printf("Warning: Failed to load texture %s: %v", filename, err)

		// Return a placeholder texture instead of failing
		placeholder := tm.createPlaceholderTexture(filename)
		return placeholder, nil
	}

	return texture, nil
}

// LoadTextureToLayer loads a texture directly to a renderer layer
func (tm *TextureManager) LoadTextureToLayer(renderer *Renderer, filename string, layer int) error {
	if layer < 0 || layer >= LayersCount {
		return fmt.Errorf("invalid layer index: %d", layer)
	}

	texture, err := tm.LoadTexture(filename)
	if err != nil {
		return err
	}

	renderer.layers[layer] = texture
	renderer.layerStates[layer].Visible = true

	log.Printf("Loaded texture %s to layer %d", filename, layer)
	return nil
}

// GetBlackTexture returns the black texture for fade effects
func (tm *TextureManager) GetBlackTexture() *ebiten.Image {
	return tm.blackTexture
}

// GetWhiteTexture returns the white texture for fade effects
func (tm *TextureManager) GetWhiteTexture() *ebiten.Image {
	return tm.whiteTexture
}

// createPlaceholderTexture creates a placeholder texture when loading fails
func (tm *TextureManager) createPlaceholderTexture(filename string) *ebiten.Image {
	img := ebiten.NewImage(tm.screenWidth, tm.screenHeight)

	// Create different colors based on filename or use default
	var fillColor [4]uint8
	lowerName := strings.ToLower(filename)

	switch {
	case strings.Contains(lowerName, "bg") || strings.Contains(lowerName, "background"):
		fillColor = [4]uint8{50, 50, 100, 255} // Dark blue
	case strings.Contains(lowerName, "title"):
		fillColor = [4]uint8{100, 50, 50, 255} // Dark red
	case strings.Contains(lowerName, "menu"):
		fillColor = [4]uint8{50, 100, 50, 255} // Dark green
	case strings.Contains(lowerName, "chip"):
		fillColor = [4]uint8{100, 100, 50, 255} // Dark yellow
	default:
		fillColor = [4]uint8{80, 80, 80, 255} // Gray
	}
	solidTexture := tm.cache.CreateSolidTexture(tm.screenWidth, tm.screenHeight, fillColor)
	img.DrawImage(solidTexture, &ebiten.DrawImageOptions{})

	return img
}

// ClearCache clears the texture cache
func (tm *TextureManager) ClearCache() {
	tm.cache.ClearCache()
}

// GetCacheSize returns the number of cached textures
func (tm *TextureManager) GetCacheSize() int {
	return tm.cache.CacheSize()
}
