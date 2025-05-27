# Regional Texture Coordinate System Analysis - SDL2 Migration Audit

## Executive Summary

The School Days visual novel engine implements a **sophisticated regional texture coordinate system** for interactive UI elements using `.glmap` files and dual texture layers. This system maps **screen regions** to **texture coordinates** for hover/click states, requiring **complete reimplementation** for SDL2 migration due to heavy OpenGL dependencies.

---

## System Architecture Overview

### **Dual-Layer Interactive System**
```
Menu System Components:
├── Base Layer: [menu].png          # Static background menu
├── Overlay Layer: [menu]_chip.png  # Interactive state graphics  
├── Region Map: [menu].glmap        # Screen coordinate regions
└── Chip Map: [menu]_chip.glmap     # Texture coordinate regions
```

**Key Concept**: Interactive regions on screen are mapped to corresponding texture coordinates in overlay textures to provide visual feedback (hover, selected, disabled states).

---

## Phase 1: .glmap File Format Analysis

### **1.1 Region Definition Format**
**Location**: `menu.cpp:377-410`

```cpp
void Menu::load_glmap(const char *name, bool is_menu) {
    Parser *map = new Parser(str);
    int count = atoi(map->get_value("Regions"));
    
    for (int i=0 ; i<count ; i++) {
        region_t *region = new region_t;
        const char *raw = map->get_value(i + 1);
        
        // Parse format: "state x1 y1 x2 y2"
        region->state = atoi(raw);          // Initial state (0=default)
        raw = strchr(raw, ' ') + 1;
        region->idx = i;                    // Region index
        region->x1 = parse_value(&raw);     // Top-left X (normalized 0.0-1.0)
        region->y1 = parse_value(&raw);     // Top-left Y (normalized 0.0-1.0)  
        region->x2 = parse_value(&raw);     // Bottom-right X (normalized 0.0-1.0)
        region->y2 = parse_value(&raw);     // Bottom-right Y (normalized 0.0-1.0)
        region->chip = NULL;                // Linked chip texture coords
    }
}
```

### **1.2 Example .glmap File Structure**
```ini
[Regions]=5
[Region1]=0 0.125 0.200 0.875 0.300
[Region2]=0 0.125 0.350 0.875 0.450  
[Region3]=0 0.125 0.500 0.875 0.600
[Region4]=0 0.125 0.650 0.875 0.750
[Region5]=0 0.125 0.800 0.875 0.900
```

**Format Details:**
- **Normalized coordinates** (0.0 to 1.0 screen space)
- **Rectangular regions** defined by two corner points
- **Initial state** (0 = default, 1 = hover, 2 = disabled, etc.)
- **Index-based** region identification

---

## Phase 2: Data Structure Analysis

### **2.1 Region Structure Definition**
**Location**: `menu.h:49-58`

```cpp
typedef struct {
    int         idx;              // Region index (0-based)
    float       x1, y1;          // Top-left corner (screen space)
    float       x2, y2;          // Bottom-right corner (screen space)
    uint32_t    state;           // Current interaction state
    region_chip_t *chip;         // Pointer to texture coordinate data
} region_t;
```

### **2.2 Chip Structure Definition** 
**Location**: `menu.h:39-47`

```cpp
typedef struct {
    uint32_t    region;          // Associated region index (1-based)
    uint32_t    state;           // State this chip represents
    float       x1, y1;          // Top-left texture coordinates
    float       x2, y2;          // Bottom-right texture coordinates  
} region_chip_t;
```

### **2.3 State Constants**
**Location**: `menu.h:32-37`

```cpp
#define MENU_DEFAULT        0    // Normal/idle state
#define MENU_MOUSE_OVER     1    // Hover state
#define MENU_DISABLE        2    // Disabled/inactive state
#define MENU_SELECTED       3    // Selected state
#define MENU_SELECTED_MOUSE 4    // Selected + hover state
```

---

## Phase 3: Chip Coordinate Loading

### **3.1 Chip Map File Loading**
**Location**: `menu.cpp:414-447`

```cpp
void Menu::load_chip(const char *name, bool is_menu) {
    Parser *map = new Parser(str);
    int count = atoi(map->get_value("Regions"));
    
    for (int i=0 ; i<count ; i++) {
        region_chip_t *region = new region_chip_t;
        const char *raw = map->get_value(i + 1);
        
        // Parse format: "region_idx state x1 y1 x2 y2"
        region->region = atoi(raw);         // Target region (1-based index)
        raw = strchr(raw, ' ') + 1;
        region->state = atoi(raw);          // State this chip represents
        raw = strchr(raw, ' ') + 1;
        region->x1 = parse_value(&raw);     // Texture coord top-left X
        region->y1 = parse_value(&raw);     // Texture coord top-left Y
        region->x2 = parse_value(&raw);     // Texture coord bottom-right X
        region->y2 = parse_value(&raw);     // Texture coord bottom-right Y
    }
}
```

### **3.2 Example Chip Map File**
```ini
[Regions]=10
[Region1]=1 1 0.000 0.000 0.500 0.200
[Region2]=1 2 0.500 0.000 1.000 0.200
[Region3]=2 1 0.000 0.200 0.500 0.400
[Region4]=2 2 0.500 0.200 1.000 0.400
[Region5]=3 1 0.000 0.400 0.500 0.600
```

**Chip Format Explanation:**
- **Region 1, State 1**: Hover texture for first button
- **Region 1, State 2**: Disabled texture for first button  
- **Region 2, State 1**: Hover texture for second button
- Each region can have **multiple chip states**

---

## Phase 4: Region-Chip Linking System

### **4.1 Dynamic Linking Process**
**Location**: `menu.cpp:87-98`

```cpp
// Called every frame in Menu::proc()
regions_chip_t::iterator chip_end = this->m_chips.end();
for (regions_chip_t::iterator it = this->m_chips.begin(); it < chip_end; it++) {
    region_chip_t *chip = *it;
    
    // Skip invalid region references
    if (chip->region > this->m_regions.size())
        continue;
        
    // Get target region (convert 1-based to 0-based index)
    region_t *region = this->m_regions.at(chip->region - 1);
    
    // Link chip if states match
    if (region->state == chip->state) {
        region->chip = chip;
    }
}
```

**Linking Logic:**
1. **Iterate all loaded chips**
2. **Find matching region** by index
3. **Compare current region state** with chip state
4. **Link chip if states match**
5. **Update every frame** for dynamic state changes

---

## Phase 5: Mouse Interaction System

### **5.1 Region Hit Testing**
**Location**: `menu.cpp:31-53`

```cpp
bool Menu::region_check(region_t *region, CURSOR *pointer, int idx) {
    // Check if mouse is within region bounds
    if ((region->x1 <= pointer->x) && (region->x2 >= pointer->x) &&
        (region->y1 <= pointer->y) && (region->y2 >= pointer->y)) {
        
        // Update state based on current state
        if (region->state == MENU_DEFAULT)
            region->state = MENU_MOUSE_OVER;
        else if (region->state == MENU_SELECTED)  
            region->state = MENU_SELECTED_MOUSE;
            
        // Handle click events
        if (pointer->left_presed) {
            pointer->left_presed = false;
            this->next_state(idx);  // Trigger menu action
            return true;
        }
    } else {
        // Mouse left region - revert state
        if (region->state == MENU_MOUSE_OVER)
            region->state = MENU_DEFAULT;
        else if (region->state == MENU_SELECTED_MOUSE)
            region->state = MENU_SELECTED;
    }
    return true;
}
```

### **5.2 Cursor Coordinate Normalization**
The system expects **normalized coordinates** (0.0 to 1.0) where:
- `pointer->x = 0.0` = Left screen edge
- `pointer->x = 1.0` = Right screen edge  
- `pointer->y = 0.0` = Top screen edge
- `pointer->y = 1.0` = Bottom screen edge

---

## Phase 6: OpenGL Rendering Integration

### **6.1 Regional Overlay Rendering**
**Location**: `interface.cpp:207-221`

```cpp
// Render interactive menu overlays
if (!global.menu->in_dlg()) {
    regions_t regions = *global.menu->get_regions();
    
    for (int i=0 ; i<regions.size() ; i++) {
        region_t *region = regions.at(i);
        region_chip_t *chip = region->chip;
        
        // Only render if region has state and linked chip
        if ((region->state != MENU_DEFAULT) && (chip)) {
            glBindTexture(GL_TEXTURE_2D, this->m_textures[LAYER_MENU_OVERLAY]);
            glBegin(GL_QUADS);
                // Map chip texture coords to region screen coords
                glTexCoord2f(chip->x1, chip->y1); glVertex3f(region->x1, region->y1, LAYER_MENU_OVERLAY);
                glTexCoord2f(chip->x2, chip->y1); glVertex3f(region->x2, region->y1, LAYER_MENU_OVERLAY);
                glTexCoord2f(chip->x2, chip->y2); glVertex3f(region->x2, region->y2, LAYER_MENU_OVERLAY);
                glTexCoord2f(chip->x1, chip->y2); glVertex3f(region->x1, region->y2, LAYER_MENU_OVERLAY);
            glEnd();
        }
    }
}
```

**Rendering Process:**
1. **Iterate active regions**
2. **Check for valid chip link**
3. **Skip default state** (no overlay needed)
4. **Bind overlay texture**
5. **Map chip texture coordinates** to region screen coordinates
6. **Render quad** with manual vertex specification

---

## Phase 7: Coordinate Mapping Tool Analysis

### **7.1 CMAP Conversion Tool**
**Location**: `cmap_conv.cpp`

The engine includes a **coordinate mapping tool** for generating `.glmap` files:

```cpp
// Automatic region detection from PNG alpha channel
void convert_chip() {
    for (int i=0 ; i<png_h ; i++) {
        for (int j=0 ; j<png_w ; j++) {
            rgba_t *pixel = (rgba_t*)png_buffer + (i*png_w + j);
            
            // Find non-transparent pixels
            if (COMPARE_PIX()) {  // pixel->a > 0
                float x = 1.0 * j / png_w;
                float y = 1.0 * i / png_h;
                
                if (!in_exist_region(x, y)) {
                    // Flood-fill to find region bounds
                    x_min = x_max = j;
                    y_min = y_max = i;
                    scan_pixel(j, i);
                    
                    // Store normalized coordinates
                    chip_regions[count].x1 = x_min * 1. / png_w;
                    chip_regions[count].y1 = y_min * 1. / png_h;
                    chip_regions[count].x2 = x_max * 1. / png_w;  
                    chip_regions[count].y2 = y_max * 1. / png_h;
                }
            }
        }
    }
}
```

**Tool Features:**
- **Automatic region detection** from PNG alpha channels
- **Flood-fill algorithm** for connected region finding
- **Coordinate normalization** to 0.0-1.0 range
- **`.glmap` file generation** for both regions and chips

---

## SDL2 Migration Strategy

### **System Replacement Requirements**

#### **1. Replace OpenGL Quad Rendering**
```cpp
// Current OpenGL approach
glBegin(GL_QUADS);
    glTexCoord2f(chip->x1, chip->y1); glVertex3f(region->x1, region->y1, z);
    // ... more vertices
glEnd();

// SDL2 replacement
SDL_Rect src_rect = {
    chip->x1 * texture_width, chip->y1 * texture_height,
    (chip->x2 - chip->x1) * texture_width, (chip->y2 - chip->y1) * texture_height
};
SDL_Rect dst_rect = {
    region->x1 * screen_width, region->y1 * screen_height,
    (region->x2 - region->x1) * screen_width, (region->y2 - region->y1) * screen_height  
};
SDL_RenderCopy(renderer, overlay_texture, &src_rect, &dst_rect);
```

#### **2. Coordinate System Updates**
- **Convert normalized coordinates** to pixel coordinates
- **Handle screen resolution scaling** 
- **Implement proper aspect ratio handling**

#### **3. State Management Simplification**
- **Maintain existing state machine** logic
- **Replace OpenGL texture binding** with SDL2 texture management
- **Simplify rendering pipeline** with `SDL_RenderCopy()`

### **Migration Complexity Assessment**

| Component | Current Complexity | SDL2 Complexity | Migration Effort |
|-----------|-------------------|-----------------|------------------|
| File Format | Medium | No Change | None |
| Data Structures | Low | No Change | None |
| Coordinate Mapping | High | Medium | Moderate |
| Rendering Pipeline | Very High | Low | High |
| State Management | Medium | Low | Low |

### **Benefits of Migration**

1. **Simplified Rendering**: Replace 20+ lines of OpenGL with 3 lines of SDL2
2. **Automatic Scaling**: SDL2 handles resolution/aspect ratio automatically  
3. **Better Performance**: Hardware-accelerated texture copying
4. **Cross-Platform**: Consistent behavior across platforms
5. **Maintainability**: Reduced code complexity and debugging overhead

### **Preservation Requirements**

The following components should be **preserved exactly**:
- **`.glmap` file format** and parsing logic
- **Region/chip data structures** 
- **State machine** and interaction logic
- **Coordinate mapping tool** functionality

Only the **OpenGL rendering backend** requires replacement with SDL2 equivalents.

---

## Implementation Priority

1. **High Priority**: Replace OpenGL quad rendering with `SDL_RenderCopy()`
2. **Medium Priority**: Update coordinate system for pixel-based calculations
3. **Low Priority**: Optimize texture management and state transitions

The regional coordinate system represents **core UI functionality** that must be **carefully preserved** during migration while **dramatically simplifying** the rendering implementation.
