# Texture Loading Pipeline Analysis - SDL2 Migration Audit

## Executive Summary

The School Days visual novel engine uses a **complex manual texture loading pipeline** built on **libpng + OpenGL** with custom file management and coordinate mapping systems. The current implementation involves approximately **200+ lines of texture-related code** across multiple files, requiring complete replacement for SDL2 migration.

---

## Current Texture Loading Architecture

### **File Structure Overview**
```
System/
├── Background/        # Scene backgrounds (.png)
├── Screen/           # System textures (Black.png, White.png)
├── Menu/             # UI menu assets
│   ├── [menu].png    # Base menu texture
│   ├── [menu]_chip.png # Interactive overlay texture
│   ├── [menu].glmap  # Region definitions
│   └── [menu]_chip.glmap # Chip texture coordinates
└── Option/           # Settings menu assets
```

---

## Phase 1: PNG File Loading Pipeline

### **1.1 File System Integration**
**Location**: `interface.cpp:306-341`

```cpp
bool Interface::load_tex(const char *name, int idx)
{
    // Automatic .png extension handling
    char fn[MAX_PATH];
    strcpy(fn, name);
    if (!strstr(name, ".png"))
        strcat(fn, ".png");

    // Custom file system access
    Stream *str = global.fs->open(fn);
    if ((str == NULL) || (str->getFileStreamHandle() == INVALID_HANDLE_VALUE)) {
        ERROR_MESSAGE("Interface: Doesn't load texture %s\n", fn);
        return false;
    }
    
    // Load entire file into memory buffer
    uint32_t sz = str->getSize();
    char buf[sz];
    str->read(buf, sz);
    delete str;

    return this->load_tex(buf, sz, idx);
}
```

**Key Features:**
- Custom `Stream` class for file access
- Automatic `.png` extension appending
- Complete file buffering in memory
- Error handling with debug logging

---

### **1.2 PNG Image Decoding**
**Location**: `interface.cpp:342-387`

```cpp
bool Interface::load_tex(const char *buf, uint32_t sz, int idx)
{
    // libpng image structure initialization
    png_image image;
    memset(&image, 0, sizeof(png_image));
    image.version = PNG_IMAGE_VERSION;
    
    if (png_image_begin_read_from_memory(&image, buf, sz)) {
        // Force RGBA format for OpenGL compatibility
        image.format = PNG_FORMAT_RGBA;
        
        // Allocate pixel buffer
        png_bytep buffer = (png_bytep)malloc(PNG_IMAGE_SIZE(image));
        
        if (png_image_finish_read(&image, NULL, buffer, 0, NULL)) {
            // OpenGL texture creation logic follows...
        }
    }
}
```

**PNG Processing Features:**
- Uses **libpng simplified API**
- Forces **PNG_FORMAT_RGBA** (32-bit RGBA)
- Dynamic memory allocation for pixel data
- Memory-based PNG reading (no file handles)

---

### **1.3 OpenGL Texture Creation**
**Location**: `interface.cpp:354-375`

```cpp
// Texture slot management
if (idx == TEXTURE_BLACK_IDX) {
    glGenTextures(1, &this->m_texture_black);
    glBindTexture(GL_TEXTURE_2D, this->m_texture_black);
} else if (idx == TEXTURE_WHITE_IDX) {
    glGenTextures(1, &this->m_texture_white);
    glBindTexture(GL_TEXTURE_2D, this->m_texture_white);
} else {
    glGenTextures(1, &this->m_textures[idx]);
    glBindTexture(GL_TEXTURE_2D, this->m_textures[idx]);
}

// OpenGL texture parameters
glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);

// Upload pixel data to GPU
glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, image.width, image.height, 
             0, GL_RGBA, GL_UNSIGNED_BYTE, buffer);
```

**OpenGL Integration:**
- **Manual texture ID management**
- **Separate special texture slots** (black/white)
- **Linear filtering** for smooth scaling
- **RGBA format** texture upload
- **No mipmap generation**

---

## Phase 2: Layer Management System

### **2.1 Texture Layer Definitions**
**Location**: `interface.h:5-15`

```cpp
#define LAYER_BG            0    // Scene backgrounds
#define LAYER_BG_OVERLAY_0  1    // Background overlay 0
#define LAYER_BG_OVERLAY_1  2    // Background overlay 1  
#define LAYER_BG_OVERLAY_2  3    // Background overlay 2
#define LAYER_TITLE_BASE    4    // Title screen base
#define LAYER_MENU          5    // Menu interface
#define LAYER_MENU_OVERLAY  6    // Interactive menu overlay
#define LAYER_SYS_BASE      7    // System UI base
#define LAYER_DLG           8    // Dialog box
#define LAYER_DLG_OVERLAY   9    // Dialog interaction overlay
#define LAYER_OVERLAY       10   // Fade/transition overlay

#define LAYERS_COUNT        10
```

### **2.2 Texture Storage Arrays**
**Location**: `interface.h:102-105`

```cpp
class Interface {
private:
    GLuint m_textures[20];        // Main layer textures
    GLuint m_texture_black;       // System black texture
    GLuint m_texture_white;       // System white texture
    // ...
};
```

**Storage Architecture:**
- **Fixed-size texture array** (20 slots)
- **Dedicated system textures** (black/white)
- **Layer-based indexing** system
- **OpenGL texture ID storage**

---

## Phase 3: Texture Usage in Rendering

### **3.1 Basic Layer Rendering**
**Location**: `interface.cpp:194-202`

```cpp
void Interface::draw() {
    // 3D setup for 2D rendering
    gluLookAt(0., 0., 21., 0., 0., 0., 0., 1., 0.);
    glColor3f(1., 1., 1.);
    
    // Layer-by-layer rendering
    draw_quad(this->m_textures[LAYER_BG], LAYER_BG);
    draw_quad(this->m_textures[LAYER_TITLE_BASE], LAYER_TITLE_BASE);
    draw_quad(this->m_textures[LAYER_MENU], LAYER_MENU);
    // ... additional layers
}
```

### **3.2 Quad Rendering Function**
**Location**: `interface.cpp:142-152`

```cpp
void draw_quad(GLuint texture, int z) {
    if (texture == uint32_t(-1)) return;
    
    glBindTexture(GL_TEXTURE_2D, texture);
    glBegin(GL_QUADS);
        glTexCoord2f(0.0f, 0.0f); glVertex3i(0, 0, z);  // Bottom-left
        glTexCoord2f(1.0f, 0.0f); glVertex3i(1, 0, z);  // Bottom-right
        glTexCoord2f(1.0f, 1.0f); glVertex3i(1, 1, z);  // Top-right
        glTexCoord2f(0.0f, 1.0f); glVertex3i(0, 1, z);  // Top-left
    glEnd();
}
```

**Rendering Features:**
- **Legacy OpenGL fixed function** (`glBegin`/`glEnd`)
- **Full texture coordinates** (0.0 to 1.0)
- **3D Z-layering** for 2D content
- **Manual vertex/texture coordinate specification**

---

## Phase 4: Texture Cleanup Pipeline

### **4.1 Texture Unloading**
**Location**: `interface.cpp:388-405`

```cpp
void Interface::unload_tex(int idx) {
    if (idx == TEXTURE_BLACK_IDX) {
        glBindTexture(GL_TEXTURE_2D, this->m_texture_black);
        glDeleteTextures(1, &this->m_texture_black);
    } else if (idx == TEXTURE_WHITE_IDX) {
        glBindTexture(GL_TEXTURE_2D, this->m_texture_white);
        glDeleteTextures(1, &this->m_texture_white);
    } else {
        glBindTexture(GL_TEXTURE_2D, this->m_textures[idx]);
        glDeleteTextures(1, &this->m_textures[idx]);
        this->m_textures[idx] = uint32_t(-1);
    }
}
```

**Cleanup Features:**
- **Manual texture binding** before deletion
- **Separate handling** for system textures
- **Array slot reset** to invalid value (-1)
- **No automatic reference counting**

---

## SDL2 Migration Requirements

### **Critical Changes Needed:**

1. **Replace libpng with SDL2_image**
   - `png_image_*` → `IMG_Load()` / `IMG_LoadTexture()`
   - Remove manual PNG decoding logic
   - Eliminate memory buffer management

2. **Replace OpenGL textures with SDL_Texture**
   - `GLuint m_textures[]` → `SDL_Texture* m_textures[]`
   - `glGenTextures()` → `SDL_CreateTextureFromSurface()`
   - `glDeleteTextures()` → `SDL_DestroyTexture()`

3. **Replace OpenGL rendering with SDL2**
   - `draw_quad()` → `SDL_RenderCopy()`
   - Remove `glBegin()`/`glEnd()` manual vertex specification
   - Replace 3D layering with `SDL_Rect` positioning

4. **Simplify texture management**
   - Eliminate manual texture binding
   - Use SDL2's automatic texture management
   - Implement proper resource cleanup

### **Performance Benefits:**
- **Hardware acceleration** through SDL2
- **Automatic texture optimization**
- **Cross-platform compatibility**
- **Simplified memory management**
- **Reduced code complexity** (~50% reduction)

---

## Migration Impact Assessment

| Component | Current Lines | SDL2 Replacement | Complexity |
|-----------|---------------|------------------|------------|
| PNG Loading | ~45 lines | ~5 lines | Simple |
| Texture Creation | ~35 lines | ~10 lines | Simple |
| Rendering Pipeline | ~60 lines | ~20 lines | Moderate |
| Cleanup Management | ~20 lines | ~5 lines | Simple |
| **Total** | **~160 lines** | **~40 lines** | **75% reduction** |

The texture loading pipeline represents a **major simplification opportunity** in the SDL2 migration, with significant code reduction and improved maintainability.
