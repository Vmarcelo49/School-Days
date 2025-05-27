Ia stuff

# SDL2 MIGRATION INSTRUCTIONS FOR AI AGENT

## TASK OVERVIEW
Replace OpenGL-based graphics system with SDL2 in the School Days visual novel engine.
Focus ONLY on core graphics replacement - Stage 2 of the migration plan.


## PRIMARY OBJECTIVE
Replace the OpenGL rendering system in interface.cpp/interface.h with SDL2 equivalents while maintaining same functionality.

## CRITICAL CONSTRAINTS
1. DO NOT modify audio, video, or input systems yet
2. DO NOT change file system or script engine
3. MAINTAIN exact same API calls from game.cpp and menu.cpp
4. PRESERVE all layer definitions and rendering order
5. KEEP regional coordinate system working identically

## FILES TO CREATE

### PRIMARY FILES (Complete Replacement)
- interface.cpp → Replace ALL OpenGL code with SDL2
- interface.h → Replace class definition with SDL2 equivalents

### SECONDARY FILES (Minor Updates)
- main.cpp → Update initialization calls only
- game.cpp → Update interface method calls if needed
- menu.cpp → Update coordinate conversion if needed

### FILES TO NOT TOUCH
- sound.cpp/sound.h (audio system - separate stage)
- video.cpp/video.h (video system - separate stage)
- All files in fs.cpp, stream.cpp, parser.cpp, settings.cpp

## SPECIFIC IMPLEMENTATION REQUIREMENTS

### 1. REPLACE OpenGL Context Management
REMOVE these OpenGL platform-specific variables from interface.h:
```cpp
// Windows
HWND m_wnd;
HGLRC m_rc;
HDC m_dc;
HINSTANCE m_instance;

// Unix  
Display *m_dpy;
Window m_root, m_win;
GLXContext m_glc;
XVisualInfo *m_vi;
```

REPLACE with SDL2 equivalents:
```cpp
SDL_Window* m_window;
SDL_Renderer* m_renderer;
```

### 2. REPLACE Texture Storage System
REMOVE OpenGL texture arrays:
```cpp
GLuint m_textures[20];
GLuint m_texture_black;
GLuint m_texture_white;
```

REPLACE with SDL2 textures:
```cpp
SDL_Texture* m_layers[LAYERS_COUNT];
SDL_Texture* m_texture_black;
SDL_Texture* m_texture_white;
```

### 3. REPLACE Core Methods

#### init_gl() → init_sdl2()
REMOVE all OpenGL initialization:
- glClearColor, glDepthFunc, glEnable(GL_DEPTH_TEST)
- glEnable(GL_TEXTURE_2D), glBlendFunc
- Matrix setup code

REPLACE with:
```cpp
bool init_sdl2(const char* title, int width, int height, bool fullscreen) {
    if (SDL_Init(SDL_INIT_VIDEO) < 0) return false;
    
    m_window = SDL_CreateWindow(title, 
                               SDL_WINDOWPOS_CENTERED, 
                               SDL_WINDOWPOS_CENTERED,
                               width, height, 
                               fullscreen ? SDL_WINDOW_FULLSCREEN : SDL_WINDOW_SHOWN);
    if (!m_window) return false;
    
    m_renderer = SDL_CreateRenderer(m_window, -1, 
                                  SDL_RENDERER_ACCELERATED | 
                                  SDL_RENDERER_PRESENTVSYNC);
    if (!m_renderer) return false;
    
    SDL_SetRenderDrawBlendMode(m_renderer, SDL_BLENDMODE_BLEND);
    return true;
}
```

#### load_tex() Method Replacement
REMOVE complex PNG + OpenGL loading pipeline (200+ lines)
REPLACE with SDL2 direct loading:
```cpp
bool load_texture(const char* name, int idx) {
    // Keep existing file system integration
    char fn[MAX_PATH];
    strcpy(fn, name);
    if (!strstr(name, ".png")) strcat(fn, ".png");
    
    Stream *str = global.fs->open(fn);
    if (!str) return false;
    
    // Use SDL2_image for direct loading
    SDL_RWops* rw = SDL_RWFromMem(buffer, size);
    SDL_Surface* surface = IMG_Load_RW(rw, 1);
    if (!surface) return false;
    
    // Store in appropriate layer slot
    if (idx == TEXTURE_BLACK_IDX) {
        m_texture_black = SDL_CreateTextureFromSurface(m_renderer, surface);
    } else if (idx == TEXTURE_WHITE_IDX) {
        m_texture_white = SDL_CreateTextureFromSurface(m_renderer, surface);
    } else {
        m_layers[idx] = SDL_CreateTextureFromSurface(m_renderer, surface);
    }
    
    SDL_FreeSurface(surface);
    return true;
}
```

#### draw() Method Replacement
REMOVE OpenGL rendering pipeline:
- glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT)
- gluLookAt() 3D camera positioning
- draw_quad() with glBegin(GL_QUADS)

REPLACE with SDL2 rendering:
```cpp
void draw() {
    SDL_SetRenderDrawColor(m_renderer, 0, 0, 0, 255);
    SDL_RenderClear(m_renderer);
    
    // Render layers in order (preserve exact same order)
    render_layer(LAYER_BG);
    render_layer(LAYER_TITLE_BASE);
    render_layer(LAYER_MENU);
    render_layer(LAYER_MENU_OVERLAY);
    render_layer(LAYER_SYS_BASE);
    render_layer(LAYER_DLG);
    render_layer(LAYER_DLG_OVERLAY);
    render_layer(LAYER_OVERLAY);
    
    SDL_RenderPresent(m_renderer);
}

void render_layer(int layer) {
    SDL_Texture* texture = m_layers[layer];
    if (!texture) return;
    
    // Full screen rendering (maintain current behavior)
    SDL_RenderCopy(m_renderer, texture, NULL, NULL);
}
```

### 4. PRESERVE Layer System Constants
KEEP these definitions EXACTLY as they are:
```cpp
#define LAYER_BG            0
#define LAYER_BG_OVERLAY_0  1
#define LAYER_BG_OVERLAY_1  2
#define LAYER_BG_OVERLAY_2  3
#define LAYER_TITLE_BASE    4
#define LAYER_MENU          5
#define LAYER_MENU_OVERLAY  6
#define LAYER_SYS_BASE      7
#define LAYER_DLG           8
#define LAYER_DLG_OVERLAY   9
#define LAYER_OVERLAY       10
#define LAYERS_COUNT        10
```

### 5. PRESERVE Regional Coordinate System
DO NOT CHANGE the regional coordinate system in menu.cpp
The .glmap file loading and region_t structures must work identically
Only coordinate conversion from normalized (0.0-1.0) to pixels may need updating

### 6. HANDLE Platform-Specific Code
REMOVE interface_win and interface_unix files completely
SDL2 handles all platform differences automatically

## TESTING REQUIREMENTS
After implementation, verify:
1. Window opens with correct title and size
2. All PNG textures load without errors
3. Layers render in correct order (background to overlay)
4. Menu regions still respond to mouse correctly
5. No OpenGL dependencies remain in code

## COMPILATION REQUIREMENTS
UPDATE build files to:
- REMOVE: opengl32.lib, glu32.lib, opengl dependencies
- ADD: SDL2.lib, SDL2main.lib, SDL2_image.lib
- REMOVE: GL/gl.h, GL/glu.h includes
- ADD: SDL.h, SDL_image.h includes

## ERROR HANDLING
Maintain existing error handling patterns:
- Use ERROR_MESSAGE() macro for errors
- Return false for failures
- Clean up SDL resources in destructor

## PERFORMANCE NOTES
SDL2 should provide equal or better performance than the deprecated OpenGL 1.x code being replaced.
The new system will be much simpler and more maintainable.

## DEBUGGING TIPS
If textures don't appear:
1. Check SDL_GetError() after each SDL call
2. Verify texture creation succeeded (not NULL)
3. Ensure renderer is created with correct flags
4. Check that layer indices match expected values

## SUCCESS CRITERIA
The engine should run identically to before but without any OpenGL dependencies.
All visual elements should appear exactly the same.
All interactive menu regions should work exactly the same.
