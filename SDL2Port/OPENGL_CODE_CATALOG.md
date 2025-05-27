# OpenGL Code Catalog - Dependency Audit

## Overview

This document catalogs all OpenGL-specific code found in the `interface.cpp` and `interface.h` files as part of Stage 1.1 of the SDL2 migration plan. This analysis identifies every piece of OpenGL-dependent code that will need to be replaced or removed during the migration.

## Executive Summary

The interface system is **heavily dependent** on OpenGL with approximately **300+ lines of OpenGL-specific code** across the interface files. The system uses both legacy OpenGL 1.x fixed-function pipeline and some newer OpenGL features, making it a prime candidate for SDL2 migration.

### **Critical Dependencies Found:**
- **Legacy OpenGL 1.x**: Uses deprecated `glBegin()/glEnd()` calls
- **Platform-specific context management**: Different code paths for Windows and Unix
- **Manual texture management**: Complex PNG loading + OpenGL texture creation
- **3D rendering for 2D content**: Unnecessary 3D positioning and depth buffering
- **Manual vertex/texture coordinate management**: Hand-coded quad rendering

---

## File Analysis

### `interface.h` - OpenGL Dependencies

#### **Preprocessor Directives**
```cpp
#ifdef USE_OPENGL
    #include <GL/gl.h>
    #ifndef _WIN32
        #include <GL/glx.h>
    #endif
    #include <GL/glu.h>
#else
    #include <GLES3/gl3.h>
    #include <GLES3/gl3ext.h>
    #include <EGL/egl.h>
#endif
```
**Impact**: Conditional compilation for OpenGL vs OpenGL ES
**SDL2 Replacement**: No conditional compilation needed, SDL2 handles all platforms

#### **Platform-Specific Context Variables**
```cpp
// Windows-specific
#ifdef _WIN32
    HWND                    m_wnd;
    HGLRC                   m_rc;          // OpenGL Rendering Context
    HDC                     m_dc;          // Device Context
    HINSTANCE               m_instance;

// Unix-specific  
#elif defined __unix__
    Display                 *m_dpy;
    Window                  m_root;
    XVisualInfo             *m_vi;
    Colormap                m_cmap;
    XSetWindowAttributes    m_swa;
    Window                  m_win;
    GLXContext              m_glc;         // GLX Context
    GLint                   m_att[5];      // GLX Attributes
#endif
```
**Impact**: Platform-specific OpenGL context management
**SDL2 Replacement**: Single `SDL_Window*` and `SDL_Renderer*` for all platforms

#### **OpenGL Texture Storage**
```cpp
GLuint                  m_textures[20];        // Main texture array
GLuint                  m_texture_black;       // System black texture
GLuint                  m_texture_white;       // System white texture
```
**Impact**: OpenGL texture IDs for layer management
**SDL2 Replacement**: `SDL_Texture* m_layers[LAYERS_COUNT]`

#### **OpenGL-Specific Methods**
```cpp
void init_gl();                                    // OpenGL initialization
void resize_gl(GLsizei width, GLsizei height);    // OpenGL viewport setup
```
**Impact**: OpenGL state management
**SDL2 Replacement**: SDL2 renderer initialization and viewport management

---

### `interface.cpp` - OpenGL Dependencies

#### **Core OpenGL Initialization (`init_gl()`)**
```cpp
void Interface::init_gl()
{
    glClearColor(0.0, 0.0, 0.0, 0.0);        // Clear color
    glDepthFunc(GL_LEQUAL);                   // Depth testing
    glEnable(GL_DEPTH_TEST);                  // Enable depth buffer
    glDepthMask(GL_TRUE);                     // Depth writing
    
    glEnable(GL_TEXTURE_2D);                  // Enable texturing
    
    glEnable(GL_BLEND);                       // Alpha blending
    glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
    
    glDisable(GL_COLOR_MATERIAL);             // Material lighting
    glPixelStorei(GL_UNPACK_ALIGNMENT, 1);    // Pixel alignment
    glTexEnvi(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE); // Texture environment
    
    // Initialize texture array
    for (int i=0 ; i<20 ; i++)
        this->m_textures[i] = uint32_t(-1);
}
```
**Impact**: Complete OpenGL state setup
**SDL2 Replacement**: SDL2 renderer creation with built-in alpha blending

#### **OpenGL Viewport Management (`resize_gl()`)**
```cpp
void Interface::resize_gl(GLsizei width, GLsizei height)
{
    glViewport(0,0,width,height);               // Set viewport
    
    glMatrixMode(GL_PROJECTION);                // Projection matrix
    glLoadIdentity();                           // Reset matrix
    glOrtho(0., 1., 1., 0., 1., 30.);          // Orthographic projection
    
    glMatrixMode(GL_MODELVIEW);                 // Model-view matrix  
    glLoadIdentity();                           // Reset matrix
}
```
**Impact**: 3D matrix setup for 2D rendering (overkill)
**SDL2 Replacement**: Automatic 2D coordinate system

#### **Legacy Quad Rendering (`draw_quad()`)**
```cpp
void draw_quad(GLuint texture, int z)
{
    if (texture == uint32_t(-1))
        return;
    glBindTexture(GL_TEXTURE_2D, texture);      // Bind texture
    glBegin(GL_QUADS);                          // Start quad (DEPRECATED)
    glTexCoord2f(0.0f, 0.0f); glVertex3i(0, 0, z);
    glTexCoord2f(1.0f, 0.0f); glVertex3i(1, 0, z);
    glTexCoord2f(1.0f, 1.0f); glVertex3i(1, 1, z);
    glTexCoord2f(0.0f, 1.0f); glVertex3i(0, 1, z);
    glEnd();                                    // End quad (DEPRECATED)
}
```
**Impact**: Uses deprecated immediate mode rendering
**SDL2 Replacement**: `SDL_RenderCopy()` for direct texture rendering

#### **Complex Layer Rendering (`draw()`)**
```cpp
void Interface::draw()
{
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);  // Clear buffers
    glLoadIdentity();                                     // Reset matrix
    
    gluLookAt(0., 0., 21., 0., 0., 0., 0., 1., 0.);     // 3D camera positioning
    
    glColor3f(1., 1., 1.);                               // Set color
    
    // Layer rendering
    draw_quad(this->m_textures[LAYER_BG], LAYER_BG);
    draw_quad(this->m_textures[LAYER_TITLE_BASE], LAYER_TITLE_BASE);
    draw_quad(this->m_textures[LAYER_MENU], LAYER_MENU);
    
    // Interactive region rendering
    if (!global.menu->in_dlg()) {
        regions_t regions = *global.menu->get_regions();
        for (int i=0 ; i<regions.size() ; i++) {
            region_t *region = regions.at(i);
            if (region->state != MENU_DEFAULT) {
                glBindTexture(GL_TEXTURE_2D, this->m_textures[LAYER_MENU_OVERLAY]);
                glBegin(GL_QUADS);  // Manual quad for region
                    glTexCoord2f(chip->x1, chip->y1); glVertex3f(region->x1, region->y1, LAYER_MENU_OVERLAY);
                    glTexCoord2f(chip->x2, chip->y1); glVertex3f(region->x2, region->y1, LAYER_MENU_OVERLAY);
                    glTexCoord2f(chip->x2, chip->y2); glVertex3f(region->x2, region->y2, LAYER_MENU_OVERLAY);
                    glTexCoord2f(chip->x1, chip->y2); glVertex3f(region->x1, region->y2, LAYER_MENU_OVERLAY);
                glEnd();
            }
        }
    }
    
    // Fade effect rendering
    if (this->m_fade_color != -1) {
        glColor4d(this->m_fade_color, this->m_fade_color, this->m_fade_color, this->m_fade);
        if (this->m_fade_color)
            draw_quad(this->m_texture_white, LAYER_OVERLAY);
        else
            draw_quad(this->m_texture_black, LAYER_OVERLAY);
    }
    
    // Platform-specific buffer swap
    #ifdef _WIN32
        SwapBuffers(this->m_dc);
    #elif defined __unix__
        glXSwapBuffers(this->m_dpy, this->m_win);
    #endif
}
```
**Impact**: Complex 3D rendering pipeline for simple 2D layering
**SDL2 Replacement**: Simple `SDL_RenderCopy()` calls with automatic layering

#### **Manual PNG Texture Loading (`load_tex()`)**
```cpp
bool Interface::load_tex(const char *buf, uint32_t sz, int idx)
{
    png_image image;
    memset(&image, 0, sizeof(png_image));
    image.version = PNG_IMAGE_VERSION;
    
    if (png_image_begin_read_from_memory(&image, buf, sz)) {
        image.format = PNG_FORMAT_RGBA;
        png_bytep buffer = (png_bytep)malloc(PNG_IMAGE_SIZE(image));
        
        if (png_image_finish_read(&image, NULL, buffer, 0, NULL)) {
            // Manual OpenGL texture creation
            if (idx == TEXTURE_BLACK_IDX) {
                glGenTextures(1, &this->m_texture_black);
                glBindTexture(GL_TEXTURE_2D, this->m_texture_black);
            } else {
                glGenTextures(1, &this->m_textures[idx]);
                glBindTexture(GL_TEXTURE_2D, this->m_textures[idx]);
            }
            
            // Set texture parameters
            glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
            glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
            
            // Upload texture data
            glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, image.width, image.height, 0, GL_RGBA, GL_UNSIGNED_BYTE, buffer);
        }
        free(buffer);
    }
    return true;
}
```
**Impact**: Complex manual texture loading with libpng + OpenGL
**SDL2 Replacement**: `IMG_Load()` + `SDL_CreateTextureFromSurface()`

#### **OpenGL Texture Cleanup (`unload_tex()`)**
```cpp
void Interface::unload_tex(int idx)
{
    if (idx == TEXTURE_BLACK_IDX) {
        glBindTexture(GL_TEXTURE_2D, this->m_texture_black);
        glDeleteTextures(1, &this->m_texture_black);
    } else {
        glBindTexture(GL_TEXTURE_2D, this->m_textures[idx]);
        glDeleteTextures(1, &this->m_textures[idx]);
        this->m_textures[idx] = uint32_t(-1);
    }
}
```
**Impact**: Manual OpenGL texture cleanup
**SDL2 Replacement**: `SDL_DestroyTexture()`

---

### Platform-Specific Files

#### **`interface_win` - Windows OpenGL Context**

**Window Class Registration:**
```cpp
wc.lpszClassName = "OpenGL";  // OpenGL-specific window class
```

**Pixel Format Descriptor:**
```cpp
static PIXELFORMATDESCRIPTOR pfd = {
    sizeof(PIXELFORMATDESCRIPTOR),
    1,
    PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL | PFD_DOUBLEBUFFER,  // OpenGL flags
    PFD_TYPE_RGBA,
    SCR_BITS,     // Color depth
    0, 0, 0, 0, 0, 0,  // Color bits
    0,            // Alpha buffer
    0,            // Shift bit
    0,            // Accumulation buffer
    0, 0, 0, 0,   // Accumulation bits
    16,           // Z-buffer depth
    0,            // Stencil buffer
    0,            // Auxiliary buffer
    PFD_MAIN_PLANE,
    0,
    0, 0, 0
};
```

**OpenGL Context Creation:**
```cpp
PixelFormat = ChoosePixelFormat(this->m_dc, &pfd);
SetPixelFormat(this->m_dc, PixelFormat, &pfd);
this->m_rc = wglCreateContext(this->m_dc);      // Create OpenGL context
wglMakeCurrent(this->m_dc, this->m_rc);         // Activate context
```

**OpenGL Context Cleanup:**
```cpp
wglMakeCurrent(NULL, NULL);                     // Release context
wglDeleteContext(this->m_rc);                   // Delete context
```

#### **`interface_unix` - Unix OpenGL Context**

**GLX Visual Selection:**
```cpp
this->m_vi = glXChooseVisual(this->m_dpy, 0, this->m_att);  // Choose visual
```

**GLX Context Creation:**
```cpp
this->m_glc = glXCreateContext(this->m_dpy, this->m_vi, NULL, GL_TRUE);  // Create context
glXMakeCurrent(this->m_dpy, this->m_win, this->m_glc);                   // Activate context
```

**GLX Context Cleanup:**
```cpp
glXMakeCurrent(this->m_dpy, None, NULL);        // Release context
glXDestroyContext(this->m_dpy, this->m_glc);    // Destroy context
```

---

## Detailed Dependency Classification

### **Category 1: Critical OpenGL Dependencies (Must Replace)**

| Component | Lines of Code | Complexity | SDL2 Replacement |
|-----------|---------------|------------|------------------|
| Context Management | ~150 lines | High | `SDL_CreateWindow()` + `SDL_CreateRenderer()` |
| Texture Loading | ~60 lines | High | `IMG_Load()` + `SDL_CreateTextureFromSurface()` |
| Rendering Pipeline | ~80 lines | High | `SDL_RenderCopy()` + `SDL_RenderPresent()` |
| Matrix Operations | ~20 lines | Medium | Not needed (2D coordinate system) |
| Depth Buffering | ~10 lines | Low | Not needed (2D layering) |

### **Category 2: Platform Abstractions (Eliminate)**

| Component | Platform | Lines of Code | SDL2 Benefit |
|-----------|----------|---------------|--------------|
| Window Creation | Windows | ~100 lines | Unified window creation |
| Window Creation | Unix | ~80 lines | Unified window creation |
| Event Handling | Windows | ~60 lines | Unified event system |
| Event Handling | Unix | ~40 lines | Unified event system |
| Context Management | Both | ~80 lines | Automatic context handling |

### **Category 3: Legacy OpenGL Features (Remove)**

| Feature | Usage | Replacement Strategy |
|---------|-------|---------------------|
| `glBegin()/glEnd()` | Quad rendering | `SDL_RenderCopy()` |
| Fixed-function pipeline | All rendering | SDL2 hardware acceleration |
| Matrix transformations | 2D positioning | Direct 2D coordinates |
| `gluLookAt()` | Camera positioning | Not needed |
| Manual depth sorting | Layer management | Render order |

---

## Migration Impact Assessment

### **Code Replacement Requirements**

1. **Complete Rewrite Required (300+ lines):**
   - `interface.cpp` - Core rendering system
   - `interface.h` - Class definition and OpenGL types
   - `interface_win` - Windows-specific code
   - `interface_unix` - Unix-specific code

2. **API Changes Required:**
   - Texture loading interface
   - Rendering method signatures  
   - Window creation parameters
   - Event handling callbacks

3. **Dependencies to Remove:**
   - OpenGL headers (`GL/gl.h`, `GL/glu.h`, `GL/glx.h`)
   - Platform-specific OpenGL libraries
   - Manual PNG loading code (replaced by SDL2_image)

### **Expected Benefits Post-Migration**

1. **Code Reduction:**
   - Remove ~180 lines of platform-specific code
   - Remove ~120 lines of OpenGL setup/teardown
   - Simplify texture loading by ~40 lines

2. **Maintenance Improvements:**
   - Single codebase for all platforms
   - No OpenGL version compatibility issues
   - Automatic hardware acceleration detection

3. **Feature Enhancements:**
   - Built-in PNG/JPG/GIF support
   - Automatic scaling and rotation
   - Better error handling and debugging

---

## Conclusion

The OpenGL code audit reveals that the interface system is **deeply integrated with OpenGL** and will require a **complete architectural rewrite** for SDL2 migration. However, the current implementation uses OpenGL in an overcomplicated way for simple 2D operations, making it an excellent candidate for SDL2's purpose-built 2D rendering capabilities.

### **Key Findings:**
- **300+ lines** of OpenGL-specific code identified
- **Heavy reliance** on deprecated OpenGL 1.x features
- **Platform-specific complexity** can be completely eliminated
- **3D rendering overhead** provides no benefit for 2D operations

### **Migration Recommendation:**
Proceed with complete SDL2 rewrite as outlined in the migration plan. The code simplification and maintenance benefits significantly outweigh the initial development effort.

This audit provides the foundation for informed decision-making throughout the SDL2 migration process.
