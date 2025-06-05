# GO EBITEN MIGRATION INSTRUCTIONS FOR AI AGENT

## TASK OVERVIEW
Replace OpenGL-based graphics system with Go Ebiten in the School Days visual novel engine.
Focus ONLY on core graphics replacement - Complete engine rewrite required due to language change.

## PRIMARY OBJECTIVE
Rewrite the entire C++ OpenGL rendering system as a Go Ebiten application while maintaining same functionality and visual appearance.

## CRITICAL CONSTRAINTS
1. PRESERVE exact visual appearance and functionality
2. MAINTAIN layer definitions and rendering order
3. PRESERVE regional coordinate system behavior
4. KEEP same game logic and menu interaction patterns
5. MAINTAIN compatibility with existing .glmap and .png asset files

## PROJECT STRUCTURE TO CREATE

### PRIMARY GO FILES (Complete Rewrite)
```
main.go              → Entry point, Ebiten game loop
interface.go         → Rendering system (replaces interface.cpp/h)
menu.go             → Menu system with region handling
assets.go           → Asset loading and management
coordinates.go      → Regional coordinate system
config.go           → Configuration and constants
```

### SUPPORTING FILES
```
go.mod              → Go module definition
assets/             → Asset directory (symlink to existing)
├── System/
├── Background/
├── Menu/
└── Option/
```

## SPECIFIC IMPLEMENTATION REQUIREMENTS

### 1. REPLACE C++ Classes with Go Structs

**Original C++ Interface Class** → **Go Interface Struct**:
```go
type Interface struct {
    // Replace OpenGL context with Ebiten renderer
    screenWidth  int
    screenHeight int
    
    // Replace OpenGL textures with Ebiten images
    layers      [LayersCount]*ebiten.Image
    textureBlack *ebiten.Image
    textureWhite *ebiten.Image
    
    // Fade system
    fadeColor  int
    fadeAmount float64
}
```

### 2. REPLACE OpenGL Rendering with Ebiten Draw

**Original OpenGL Rendering:**
```cpp
glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
gluLookAt(0., 0., 21., 0., 0., 0., 0., 1., 0.);
draw_quad(this->m_textures[LAYER_BG], LAYER_BG);
SwapBuffers(this->m_dc);
```

**Ebiten Replacement:**
```go
func (i *Interface) Draw(screen *ebiten.Image) {
    // Clear screen (automatic in Ebiten)
    
    // Render layers in order (preserve exact same order)
    i.renderLayer(screen, LayerBG)
    i.renderLayer(screen, LayerTitleBase)
    i.renderLayer(screen, LayerMenu)
    i.renderLayer(screen, LayerMenuOverlay)
    i.renderLayer(screen, LayerSysBase)
    i.renderLayer(screen, LayerDlg)
    i.renderLayer(screen, LayerDlgOverlay)
    i.renderLayer(screen, LayerOverlay)
    
    // Handle fade effects
    i.renderFade(screen)
}

func (i *Interface) renderLayer(screen *ebiten.Image, layer int) {
    if i.layers[layer] == nil {
        return
    }
    
    opts := &ebiten.DrawImageOptions{}
    // Scale to screen size if needed
    screen.DrawImage(i.layers[layer], opts)
}
```

### 3. REPLACE PNG Loading with Ebiten Image Loading

**Original libpng + OpenGL:**
```cpp
png_image image;
png_image_begin_read_from_memory(&image, buf, sz);
glGenTextures(1, &texture);
glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, width, height, 0, GL_RGBA, GL_UNSIGNED_BYTE, buffer);
```

**Ebiten Replacement:**
```go
func (i *Interface) LoadTexture(filename string, layerIdx int) error {
    // Use custom file system if available, otherwise standard file loading
    data, err := assets.LoadFile(filename)
    if err != nil {
        return fmt.Errorf("failed to load %s: %v", filename, err)
    }
    
    // Decode image
    img, _, err := image.Decode(bytes.NewReader(data))
    if err != nil {
        return fmt.Errorf("failed to decode %s: %v", filename, err)
    }
    
    // Convert to Ebiten image
    ebitenImg := ebiten.NewImageFromImage(img)
    
    // Store in appropriate layer
    switch layerIdx {
    case TextureBlackIdx:
        i.textureBlack = ebitenImg
    case TextureWhiteIdx:
        i.textureWhite = ebitenImg
    default:
        i.layers[layerIdx] = ebitenImg
    }
    
    return nil
}
```

### 4. REPLACE C++ Regional Coordinate System with Go

**Original C++ Structures:**
```cpp
typedef struct {
    int idx;
    float x1, y1, x2, y2;
    uint32_t state;
    region_chip_t *chip;
} region_t;
```

**Go Equivalent:**
```go
type Region struct {
    Idx   int
    X1, Y1, X2, Y2 float64  // Normalized coordinates (0.0-1.0)
    State uint32
    Chip  *RegionChip
}

type RegionChip struct {
    Region int
    State  uint32
    X1, Y1, X2, Y2 float64  // Texture coordinates (0.0-1.0)
}

type Menu struct {
    regions []*Region
    chips   []*RegionChip
}
```

### 5. REPLACE C++ .glmap Parsing with Go

**Original C++ Parser:**
```cpp
Parser *map = new Parser(str);
int count = atoi(map->get_value("Regions"));
region->x1 = parse_value(&raw);
```

**Go Replacement:**
```go
func (m *Menu) LoadGLMap(filename string) error {
    data, err := assets.LoadFile(filename)
    if err != nil {
        return err
    }
    
    // Simple INI-style parser
    lines := strings.Split(string(data), "\n")
    var regionCount int
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "[Regions]=") {
            regionCount, _ = strconv.Atoi(strings.TrimPrefix(line, "[Regions]="))
        } else if strings.HasPrefix(line, "[Region") {
            // Parse: "0 0.125 0.200 0.875 0.300"
            parts := strings.Fields(strings.Split(line, "]=")[1])
            if len(parts) >= 5 {
                region := &Region{
                    State: parseUint32(parts[0]),
                    X1:    parseFloat64(parts[1]),
                    Y1:    parseFloat64(parts[2]),
                    X2:    parseFloat64(parts[3]),
                    Y2:    parseFloat64(parts[4]),
                }
                m.regions = append(m.regions, region)
            }
        }
    }
    return nil
}
```

### 6. IMPLEMENT Ebiten Game Interface

**Required Ebiten Methods:**
```go
type Game struct {
    interface_ *Interface
    menu      *Menu
    // ... other systems
}

func (g *Game) Update() error {
    // Handle input
    g.menu.HandleInput()
    
    // Update game logic
    g.updateGameLogic()
    
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    g.interface_.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
    return 1024, 768  // Or whatever resolution the game uses
}

func main() {
    ebiten.SetWindowSize(1024, 768)
    ebiten.SetWindowTitle("School Days")
    
    game := &Game{
        interface_: NewInterface(),
        menu:      NewMenu(),
    }
    
    // Load initial assets
    game.interface_.LoadTexture("System/Screen/Black.png", TextureBlackIdx)
    game.interface_.LoadTexture("System/Screen/White.png", TextureWhiteIdx)
    
    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}
```

### 7. PRESERVE Layer System Constants

**Keep these definitions EXACTLY as they are:**
```go
const (
    LayerBG           = 0
    LayerBGOverlay0   = 1
    LayerBGOverlay1   = 2
    LayerBGOverlay2   = 3
    LayerTitleBase    = 4
    LayerMenu         = 5
    LayerMenuOverlay  = 6
    LayerSysBase      = 7
    LayerDlg          = 8
    LayerDlgOverlay   = 9
    LayerOverlay      = 10
    LayersCount       = 10
)

const (
    MenuDefault      = 0
    MenuMouseOver    = 1
    MenuDisable      = 2
    MenuSelected     = 3
    MenuSelectedMouse = 4
)
```

### 8. HANDLE Mouse Input with Ebiten

**Replace Windows/Unix mouse handling:**
```go
func (m *Menu) HandleInput() {
    // Get mouse position
    mouseX, mouseY := ebiten.CursorPosition()
    
    // Convert to normalized coordinates (0.0-1.0)
    screenW, screenH := ebiten.WindowSize()
    normalizedX := float64(mouseX) / float64(screenW)
    normalizedY := float64(mouseY) / float64(screenH)
    
    // Check regions
    for i, region := range m.regions {
        m.checkRegion(region, normalizedX, normalizedY, i)
    }
}

func (m *Menu) checkRegion(region *Region, mouseX, mouseY float64, idx int) {
    // Check if mouse is within region bounds
    if mouseX >= region.X1 && mouseX <= region.X2 &&
       mouseY >= region.Y1 && mouseY <= region.Y2 {
        
        // Update state based on current state
        if region.State == MenuDefault {
            region.State = MenuMouseOver
        } else if region.State == MenuSelected {
            region.State = MenuSelectedMouse
        }
        
        // Handle click events
        if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
            m.nextState(idx)  // Trigger menu action
        }
    } else {
        // Mouse left region - revert state
        if region.State == MenuMouseOver {
            region.State = MenuDefault
        } else if region.State == MenuSelectedMouse {
            region.State = MenuSelected
        }
    }
}
```

## COMPILATION REQUIREMENTS

**go.mod file:**
```go
module school-days

go 1.21

require (
    github.com/hajimehoshi/ebiten/v2 v2.6.0
)
```

**Build command:**
```bash
go mod tidy
go build -o school-days.exe .
```

## ASSET COMPATIBILITY

**Preserve existing assets:**
- Keep all .png files in same structure
- Keep all .glmap files with same format
- Ensure proper file path handling for assets

**File loading system:**
```go
// Implement custom asset loader if needed to match original file system
type AssetLoader struct {
    basePath string
}

func (a *AssetLoader) LoadFile(filename string) ([]byte, error) {
    // Add .png extension if missing (match original behavior)
    if !strings.Contains(filename, ".png") {
        filename += ".png"
    }
    
    fullPath := filepath.Join(a.basePath, filename)
    return os.ReadFile(fullPath)
}
```

## TESTING REQUIREMENTS

After implementation, verify:
1. Window opens with correct title and size
2. All PNG textures load without errors
3. Layers render in correct order (background to overlay)
4. Menu regions respond to mouse correctly
5. .glmap files parse correctly
6. Interactive overlays display properly
7. State transitions work identically to original

## ERROR HANDLING

**Implement Go-style error handling:**
```go
func (i *Interface) LoadTexture(filename string, layerIdx int) error {
    if data, err := assets.LoadFile(filename); err != nil {
        return fmt.Errorf("interface: failed to load texture %s: %v", filename, err)
    }
    // ... rest of loading logic
    return nil
}
```

## PERFORMANCE NOTES

Ebiten provides:
- Hardware acceleration automatically
- Efficient 2D rendering
- Cross-platform compatibility
- Simple texture management
- Built-in input handling

The Go + Ebiten implementation should provide equal or better performance than the original OpenGL 1.x code while being much simpler and more maintainable.

## SUCCESS CRITERIA

The engine should:
1. Run identically to the original C++ version
2. Display all visual elements exactly the same
3. Handle all menu interactions identically
4. Load and display all existing assets correctly
5. Provide smooth 60fps rendering
6. Work cross-platform (Windows, Linux, macOS)

The complete rewrite in Go + Ebiten represents a significant modernization while preserving all original functionality and visual appearance.
