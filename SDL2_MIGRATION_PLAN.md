# School Days Engine - SDL2 Migration Plan

## Overview

This document outlines a comprehensive plan to migrate the School Days visual novel engine from OpenGL to SDL2, simplifying the graphics architecture while maintaining all current functionality.

## Why Migrate to SDL2?

### **Current OpenGL Overkill Analysis**
Based on the OpenGL analysis, the current engine uses 3D graphics capabilities for purely 2D operations:

- **Unnecessary Complexity**: Using 3D coordinates, depth buffering, and camera positioning for flat 2D layers
- **Legacy Dependencies**: Relies on deprecated OpenGL 1.x fixed-function pipeline
- **Platform Complexity**: Requires different OpenGL context setup for Windows/Linux
- **Limited Benefits**: No actual 3D rendering justifies the complexity

### **SDL2 Advantages for This Project**
- **Perfect Match**: SDL2 is designed exactly for 2D graphics and multimedia applications
- **Simplified API**: Direct 2D rendering without 3D concepts
- **Better Integration**: Built-in support for PNG loading, audio, input, and windowing
- **Cross-Platform**: Single API works identically across Windows, Linux, macOS
- **Active Development**: Modern, actively maintained library
- **Smaller Dependencies**: Reduces external library requirements

## Current System Analysis

### **Existing Layer System (OpenGL)**
```cpp
#define LAYER_BG            0    // Background
#define LAYER_BG_OVERLAY_0  1    // Background overlays (3 layers)
#define LAYER_BG_OVERLAY_1  2
#define LAYER_BG_OVERLAY_2  3
#define LAYER_TITLE_BASE    4    // Title screen base
#define LAYER_MENU          5    // Menu graphics
#define LAYER_MENU_OVERLAY  6    // Interactive menu elements
#define LAYER_SYS_BASE      7    // System interface
#define LAYER_DLG           8    // Dialog boxes
#define LAYER_DLG_OVERLAY   9    // Dialog overlays
#define LAYER_OVERLAY       10   // Top-level overlays (fade effects)
```

### **Current Rendering Pipeline**
1. OpenGL context creation and setup
2. Texture loading via libpng + manual OpenGL texture creation
3. 3D camera positioning for 2D layering
4. Manual quad rendering with `glBegin(GL_QUADS)`
5. Regional texture mapping for UI interactions
6. Manual alpha blending and depth sorting

### **Current Dependencies**
- **OpenGL**: `opengl32.dll` (Windows), Mesa (Linux)
- **GLU**: Camera and utility functions
- **libpng**: PNG image decoding
- **Platform-specific**: Win32 API, X11/GLX

## SDL2 Migration Architecture

### **Proposed SDL2 System**

#### **Core SDL2 Components**
```cpp
// Main SDL2 subsystems
SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO | SDL_INIT_EVENTS);

// Window and renderer creation
SDL_Window* window = SDL_CreateWindow("School Days", 
                                      SDL_WINDOWPOS_CENTERED, 
                                      SDL_WINDOWPOS_CENTERED,
                                      width, height, 
                                      SDL_WINDOW_SHOWN);

SDL_Renderer* renderer = SDL_CreateRenderer(window, -1, 
                                           SDL_RENDERER_ACCELERATED | 
                                           SDL_RENDERER_PRESENTVSYNC);
```

#### **Simplified Layer Management**
```cpp
class SDL2Interface {
private:
    SDL_Renderer* m_renderer;
    SDL_Texture* m_layers[LAYERS_COUNT];
    LayerProperties m_layer_props[LAYERS_COUNT];
    
public:
    // Direct texture loading from file
    bool load_texture(const char* filename, int layer);
    
    // Simple 2D rendering
    void render_layer(int layer, SDL_Rect* src, SDL_Rect* dst);
    
    // Built-in alpha blending
    void set_layer_alpha(int layer, uint8_t alpha);
    
    // Automatic scaling and positioning
    void set_layer_transform(int layer, float x, float y, float scale);
};
```

#### **Texture Management Simplification**
```cpp
// Replace complex OpenGL texture loading
bool SDL2Interface::load_texture(const char* filename, int layer) {
    // SDL2 handles PNG loading automatically
    SDL_Surface* surface = IMG_Load(filename);
    if (!surface) return false;
    
    // Direct texture creation
    m_layers[layer] = SDL_CreateTextureFromSurface(m_renderer, surface);
    SDL_FreeSurface(surface);
    
    // Enable alpha blending
    SDL_SetTextureBlendMode(m_layers[layer], SDL_BLENDMODE_BLEND);
    
    return m_layers[layer] != nullptr;
}
```

## Migration Plan Stages

### **Stage 1: Preparation and Analysis (Week 1)**

#### **1.1 Dependency Audit**
- [ ] Catalog all OpenGL-specific code in `interface.cpp`/`interface.h`
- [ ] Identify platform-specific rendering code (`interface_win`, `interface_unix`)
- [ ] Document current texture loading pipeline
- [ ] Map regional texture coordinate system usage

#### **1.2 SDL2 Environment Setup**
- [ ] Install SDL2 development libraries
- [ ] Install SDL2_image for PNG support
- [ ] Install SDL2_mixer for audio (replace OpenAL)
- [ ] Create test SDL2 project with basic window creation

#### **1.3 Code Structure Analysis**
```bash
# Files requiring major changes:
interface.cpp      # Complete rewrite
interface.h        # API redesign
interface_win      # Remove platform-specific code
interface_unix     # Remove platform-specific code

# Files requiring minor changes:
game.cpp          # Update interface calls
menu.cpp          # Update rendering calls
script_engine.cpp # Update graphics events

# Files potentially unchanged:
fs.cpp            # File system remains same
settings.cpp      # Configuration unchanged
stream.cpp        # Data streaming unchanged
```

### **Stage 2: Core Interface Replacement (Week 2-3)**

#### **2.1 New SDL2Interface Class Design**
```cpp
class SDL2Interface {
private:
    SDL_Window* m_window;
    SDL_Renderer* m_renderer;
    SDL_Texture* m_layers[LAYERS_COUNT];
    
    // Layer properties
    struct LayerState {
        bool visible;
        uint8_t alpha;
        SDL_Rect src_rect;
        SDL_Rect dst_rect;
        float scale_x, scale_y;
    } m_layer_states[LAYERS_COUNT];
    
    // Fade effect state
    SDL_Texture* m_fade_texture;
    uint8_t m_fade_alpha;
    bool m_fade_to_white;
    
    // Regional interaction system
    std::vector<InteractiveRegion> m_regions;
    
public:
    // Core functionality
    bool init(const char* title, int width, int height, bool fullscreen);
    void shutdown();
    
    // Texture management
    bool load_texture(const char* filename, int layer);
    void unload_texture(int layer);
    
    // Rendering
    void clear();
    void render_layers();
    void present();
    
    // Layer control
    void set_layer_visible(int layer, bool visible);
    void set_layer_alpha(int layer, uint8_t alpha);
    void set_layer_position(int layer, int x, int y);
    void set_layer_scale(int layer, float scale_x, float scale_y);
    
    // Regional interactions
    void add_interactive_region(int layer, SDL_Rect region, int state_id);
    void clear_regions(int layer);
    int get_region_at_point(int x, int y);
    
    // Fade effects
    void set_fade(uint8_t alpha, bool to_white = false);
    
    // Input handling
    void handle_mouse_motion(int x, int y);
    bool handle_mouse_click(int x, int y);
};
```

#### **2.2 Implementation Priority**
1. **Basic window creation and texture loading**
2. **Layer rendering system**
3. **Alpha blending and fade effects**
4. **Interactive region mapping**
5. **Input event handling integration**

#### **2.3 Backwards Compatibility Layer**
```cpp
// Temporary compatibility functions during migration
namespace CompatLayer {
    // Map old OpenGL calls to SDL2
    inline bool load_tex(const char* name, int idx) {
        return global.sdl_interface->load_texture(name, idx);
    }
    
    inline void draw() {
        global.sdl_interface->render_layers();
        global.sdl_interface->present();
    }
    
    inline void set_fade(double value) {
        global.sdl_interface->set_fade((uint8_t)(value * 255));
    }
}
```

### **Stage 3: Audio System Integration (Week 4)**

#### **3.1 Replace OpenAL with SDL2_mixer**
Current audio components to replace:
- `sound.cpp`/`sound.h` - Basic audio playback
- `sound_stream.cpp`/`sound_stream.h` - Streaming audio
- `video_stream.cpp` - Audio in video playback

#### **3.2 New Audio Architecture**
```cpp
class SDL2AudioManager {
private:
    Mix_Music* m_background_music;
    std::map<int, Mix_Chunk*> m_sound_effects;
    std::map<int, Mix_Chunk*> m_system_sounds;
    
public:
    bool init();
    void shutdown();
    
    // Background music
    bool load_bgm(const char* filename);
    void play_bgm(int loops = -1);
    void stop_bgm();
    void set_bgm_volume(int volume);
    
    // Sound effects
    bool load_sound_effect(const char* filename, int id);
    void play_sound_effect(int id);
    
    // System sounds
    void play_system_sound(SystemSound sound);
};
```

#### **3.3 Integration Points**
- Replace OpenAL initialization in main.cpp
- Update script engine audio event handling
- Migrate audio streaming for video playback

### **Stage 4: Input and Event System (Week 5)**

#### **4.1 SDL2 Event Integration**
```cpp
class SDL2EventManager {
private:
    SDL_Event m_event;
    bool m_running;
    
public:
    void process_events();
    bool should_quit() const { return !m_running; }
    
    // Input state
    bool is_key_pressed(SDL_Scancode key);
    void get_mouse_state(int* x, int* y, uint32_t* buttons);
    
    // Event callbacks
    void set_mouse_click_handler(std::function<void(int, int, int)> handler);
    void set_key_handler(std::function<void(SDL_Scancode, bool)> handler);
    void set_window_event_handler(std::function<void(SDL_WindowEvent)> handler);
};
```

#### **4.2 Menu System Integration**
Update `menu.cpp` to use SDL2 coordinate system:
```cpp
// Replace OpenGL normalized coordinates (0-1) with pixel coordinates
void Menu::update_regions_for_sdl2() {
    for (auto& region : m_regions) {
        // Convert from normalized to pixel coordinates
        region->x1 = region->x1 * screen_width;
        region->y1 = region->y1 * screen_height;
        region->x2 = region->x2 * screen_width;
        region->y2 = region->y2 * screen_height;
    }
}
```

### **Stage 5: Video Integration (Week 6)**

#### **5.1 SDL2 Video Playback**
Current video system uses FFmpeg with OpenGL textures. Migrate to SDL2:

```cpp
class SDL2VideoPlayer {
private:
    SDL_Texture* m_video_texture;
    AVCodecContext* m_codec_context;  // Keep FFmpeg for decoding
    SwsContext* m_sws_context;
    
public:
    bool load_video(const char* filename);
    bool update_frame();  // Decode frame to SDL2 texture
    void render(SDL_Rect* dst_rect);
    void close();
};
```

#### **5.2 Integration Strategy**
- Keep FFmpeg for video/audio decoding
- Replace OpenGL texture upload with SDL2 texture creation
- Integrate with layer system for video overlays

### **Stage 6: Build System Updates (Week 7)**

#### **6.1 CodeBlocks Project Updates**
Update `SD_win.cbp` and `SD_unix.cbp`:

**Remove Dependencies:**
- OpenGL libraries (`opengl32`, `glu32`)
- Platform-specific OpenGL headers
- Manual PNG linking

**Add Dependencies:**
```xml
<!-- Windows (SDL2_win.cbp) -->
<Add library="SDL2" />
<Add library="SDL2main" />
<Add library="SDL2_image" />
<Add library="SDL2_mixer" />

<!-- Linux (SDL2_unix.cbp) -->
<Add library="SDL2" />
<Add library="SDL2_image" />
<Add library="SDL2_mixer" />
```

#### **6.2 Cross-Platform Compilation**
```bash
# Windows (MinGW)
g++ -o schooldays *.cpp -lSDL2 -lSDL2_image -lSDL2_mixer -lavcodec -lavformat

# Linux
g++ -o schooldays *.cpp $(pkg-config --libs sdl2 SDL2_image SDL2_mixer) -lavcodec -lavformat

# macOS
clang++ -o schooldays *.cpp -framework SDL2 -framework SDL2_image -framework SDL2_mixer
```

### **Stage 7: Testing and Optimization (Week 8)**

#### **7.1 Functionality Testing**
- [ ] All menu systems work correctly
- [ ] Texture loading and display functions
- [ ] Audio playback (BGM, SFX, system sounds)
- [ ] Video playback integration
- [ ] Save/load system compatibility
- [ ] Script engine integration
- [ ] Cross-platform compatibility

#### **7.2 Performance Testing**
- [ ] Measure texture loading times
- [ ] Compare rendering performance vs OpenGL
- [ ] Memory usage analysis
- [ ] Startup time comparison

#### **7.3 Regression Testing**
- [ ] All existing game functionality preserved
- [ ] Visual quality maintained
- [ ] Audio synchronization
- [ ] Input responsiveness

## Implementation Details

### **File Changes Summary**

#### **Files to Replace Completely**
```
interface.cpp     → sdl2_interface.cpp
interface.h       → sdl2_interface.h
interface_win     → (removed, SDL2 handles platform abstraction)
interface_unix    → (removed, SDL2 handles platform abstraction)
sound.cpp         → sdl2_audio.cpp
sound.h           → sdl2_audio.h
```

#### **Files to Modify**
```
main.cpp          → Update initialization calls
game.cpp          → Replace interface calls
menu.cpp          → Update coordinate system and rendering
script_engine.cpp → Update graphics/audio event handling
video.cpp         → Integrate SDL2 video textures
```

#### **Files Unchanged**
```
fs.cpp/fs.h           → File system abstraction remains
stream.cpp/stream.h   → Data streaming unchanged
settings.cpp/settings.h → Configuration system unchanged
parser.cpp/parser.h   → Script parsing unchanged
gpk.cpp/gpk.h         → Archive system unchanged
```

### **API Migration Examples**

#### **Texture Loading**
```cpp
// Before (OpenGL)
bool Interface::load_tex(const char* name, int idx) {
    // Complex PNG loading + manual OpenGL texture creation
    Stream *str = global.fs->open(name);
    png_image image;
    // ... 50+ lines of PNG decoding
    glGenTextures(1, &texture_id);
    glBindTexture(GL_TEXTURE_2D, texture_id);
    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, width, height, 0, GL_RGBA, GL_UNSIGNED_BYTE, buffer);
}

// After (SDL2)
bool SDL2Interface::load_texture(const char* name, int idx) {
    std::string path = global.fs->get_full_path(name);
    SDL_Surface* surface = IMG_Load(path.c_str());
    if (!surface) return false;
    
    m_layers[idx] = SDL_CreateTextureFromSurface(m_renderer, surface);
    SDL_FreeSurface(surface);
    SDL_SetTextureBlendMode(m_layers[idx], SDL_BLENDMODE_BLEND);
    
    return m_layers[idx] != nullptr;
}
```

#### **Rendering**
```cpp
// Before (OpenGL)
void Interface::draw() {
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
    glLoadIdentity();
    gluLookAt(0., 0., 21., 0., 0., 0., 0., 1., 0.);
    
    for (int i = 0; i < LAYERS_COUNT; i++) {
        draw_quad(m_textures[i], i);  // Manual quad rendering
    }
    
    SwapBuffers(m_dc);
}

// After (SDL2)
void SDL2Interface::render() {
    SDL_SetRenderDrawColor(m_renderer, 0, 0, 0, 255);
    SDL_RenderClear(m_renderer);
    
    for (int i = 0; i < LAYERS_COUNT; i++) {
        if (m_layers[i] && m_layer_states[i].visible) {
            SDL_SetTextureAlphaMod(m_layers[i], m_layer_states[i].alpha);
            SDL_RenderCopy(m_renderer, m_layers[i], 
                          &m_layer_states[i].src_rect, 
                          &m_layer_states[i].dst_rect);
        }
    }
    
    SDL_RenderPresent(m_renderer);
}
```

## Expected Benefits

### **Code Simplification**
- **Reduce codebase**: Eliminate ~300 lines of OpenGL setup code
- **Single API**: Replace platform-specific rendering with unified SDL2
- **Built-in features**: PNG loading, audio mixing, input handling included

### **Improved Maintainability**
- **Modern library**: SDL2 is actively maintained and well-documented
- **Better debugging**: SDL2 provides clear error messages and debugging tools
- **Simplified building**: Fewer dependencies and linking requirements

### **Enhanced Portability**
- **Additional platforms**: Easy porting to macOS, mobile platforms, web (Emscripten)
- **Consistent behavior**: Identical rendering across all platforms
- **Future-proof**: SDL2 continues to be updated for new platforms

### **Performance Benefits**
- **Hardware acceleration**: SDL2 automatically uses GPU acceleration when available
- **Optimized 2D**: Purpose-built for 2D rendering, no 3D overhead
- **Better memory management**: Automatic texture management and optimization

## Risk Assessment

### **Low Risk**
- **Functionality preservation**: All current features can be maintained
- **Visual quality**: SDL2 supports same PNG textures and alpha blending
- **Performance**: Expected equal or better performance for 2D operations

### **Medium Risk**
- **Development time**: Estimated 8 weeks for complete migration
- **Testing requirements**: Comprehensive testing needed across platforms
- **Learning curve**: Team needs to learn SDL2 API (though simpler than OpenGL)

### **Mitigation Strategies**
- **Incremental migration**: Maintain compatibility layer during transition
- **Parallel development**: Keep OpenGL version working while developing SDL2
- **Comprehensive testing**: Thorough testing at each stage
- **Rollback plan**: Ability to revert to OpenGL version if critical issues arise

## Conclusion

Migrating from OpenGL to SDL2 is a logical evolution for the School Days engine. The current OpenGL implementation is overcomplicated for the actual rendering needs, and SDL2 provides a more appropriate, simpler, and more maintainable solution.

The migration can be completed incrementally over 8 weeks with minimal risk to existing functionality while providing significant long-term benefits in maintainability, portability, and development efficiency.

**Recommendation**: Proceed with SDL2 migration following the staged approach outlined above.
