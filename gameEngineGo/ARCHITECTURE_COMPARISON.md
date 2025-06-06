# School Days Engine: C++ vs Go Architecture Comparison

## Executive Summary

This document provides a comprehensive top-down comparison between the original C++ School Days engine and the Go port implementation, identifying missing components and architectural gaps.

## 🏗️ Overall Architecture

### C++ Original Structure
```
Game (main coordinator)
├── FS (Filesystem + GPK archives)
├── Settings (INI configuration system)
├── Interface (OpenGL graphics + windowing)
├── ScriptEngine (Event-driven timeline system)
├── Sound (OpenAL audio system)
├── Video (FFmpeg video playback)
├── Menu (State machine + clickable regions)
└── Parser (Generic INI/config parser)
```

### Go Current Structure
```
main.go (entry point)
├── filesystem/ (GPK archives) ✅
├── settings/ (JSON + INI config) ✅
├── graphics/ (Ebiten renderer) ✅
├── script/ (Event system) ✅
├── audio/ (stub) ⚠️
├── menu/ (State machine + GLMAP) ✅
├── input/ (basic input) ✅
└── engine/ (game coordinator) ⚠️
```

## 📋 Component-by-Component Analysis

### ✅ **COMPLETE**: Filesystem (FS)
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| GPK Archive Loading | ✅ | ✅ | Complete |
| File Resolution Priority | ✅ | ✅ | Complete |
| Path Normalization | ✅ | ✅ | Complete |
| Multiple Archive Support | ✅ | ✅ | Complete |

**Missing**: None - filesystem is fully functional

---

### ✅ **COMPLETE**: Settings System
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| INI File Parsing | ✅ | ✅ | Complete |
| Nested Config Loading | ✅ | ✅ | Complete |
| JSON Config Support | ❌ | ✅ | Enhanced |

**Missing**: None - settings system is enhanced beyond C++ version

---

### ⚠️ **PARTIAL**: Graphics System (Interface)
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| Layer-based Rendering | ✅ | ✅ | Complete |
| Texture Loading | ✅ | ✅ | Complete |
| Fade Effects | ✅ | ✅ | Complete |
| OpenGL/Windowing | ✅ | ✅ (Ebiten) | Complete |
| **MISSING FEATURES** | | | |
| Video Texture Playback | ✅ | ❌ | **MISSING** |
| Dynamic Texture Updates | ✅ | ❌ | **MISSING** |
| Advanced Blending Modes | ✅ | ❌ | **MISSING** |

---

### ✅ **COMPLETE**: Script Engine
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| Event Timeline | ✅ | ✅ | Complete |
| Fade Calculations | ✅ | ✅ | Complete |
| State Management | ✅ | ✅ | Complete |
| Time-based Events | ✅ | ✅ | Complete |
| **MISSING FEATURES** | | | |
| Script File Parsing | ✅ | ❌ | **MISSING** |
| Resource Memory Management | ✅ | ❌ | **MISSING** |

---

### ❌ **MISSING**: Audio System (Sound)
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| OpenAL Integration | ✅ | ❌ | **MISSING** |
| OGG Vorbis Support | ✅ | ❌ | **MISSING** |
| BGM Streaming | ✅ | ❌ | **MISSING** |
| SE (Sound Effects) | ✅ | ❌ | **MISSING** |
| Voice Playback | ✅ | ❌ | **MISSING** |
| Volume Controls | ✅ | ❌ | **MISSING** |
| Audio Stream Management | ✅ | ❌ | **MISSING** |

---

### ❌ **MISSING**: Video System
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| FFmpeg Integration | ✅ | ❌ | **MISSING** |
| Video Decoding | ✅ | ❌ | **MISSING** |
| Video-to-Texture Rendering | ✅ | ❌ | **MISSING** |
| Synchronized Playback | ✅ | ❌ | **MISSING** |

---

### ✅ **COMPLETE**: Menu System
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| State Machine | ✅ | ✅ | Complete |
| GLMAP Region Loading | ✅ | ✅ | Complete |
| Click Detection | ✅ | ✅ | Complete |
| Menu Transitions | ✅ | ✅ | Complete |

---

### ⚠️ **PARTIAL**: Game Coordination
| Feature | C++ | Go | Status |
|---------|-----|----|----|
| System Initialization | ✅ | ✅ | Complete |
| Main Game Loop | ✅ | ⚠️ | Basic |
| **MISSING FEATURES** | | | |
| System Integration | ✅ | ❌ | **MISSING** |
| Global State Management | ✅ | ❌ | **MISSING** |
| Event Coordination | ✅ | ❌ | **MISSING** |

## 🚨 Critical Missing Components

### 1. **Audio System** (HIGH PRIORITY)
The C++ version has a comprehensive audio system with:
- **OpenAL** for 3D positional audio
- **OGG Vorbis** codec support  
- **Streaming audio** for long BGM tracks
- **Multiple audio channels** (BGM, SE, Voice)
- **Volume mixing and controls**

**Required for Go**: 
- Audio library integration (suggest: [Beep](https://github.com/faiface/beep) or [Oto](https://github.com/hajimehoshi/oto))
- OGG/Vorbis decoder
- Audio stream management
- Volume controls matching settings

### 2. **Video Playback System** (HIGH PRIORITY)
The C++ version supports:
- **FFmpeg-based** video decoding
- **Video-to-OpenGL texture** rendering
- **Synchronized audio/video** playback

**Required for Go**:
- Video decoder (FFmpeg Go bindings or native decoder)
- Video-to-Ebiten texture pipeline
- Audio/video synchronization

### 3. **Script File Parser** (MEDIUM PRIORITY)
The C++ version parses script files with:
- **Tab-separated values** format
- **Event type recognition** (CreateBG, BlackFade, PlayBgm, etc.)
- **Timeline synchronization**

**Required for Go**:
- Script file format parser
- Event factory system
- File-to-event conversion

### 4. **System Integration Layer** (HIGH PRIORITY)
The C++ version has a central `global` structure that coordinates all systems:

```cpp
typedef struct {
    Game        *game;
    Interface   *_interface;
    FS          *fs;
    Settings    *settings;
    ScriptEngine*engine;
    Sound       *sound;
    Video       *video;
    Menu        *menu;
} global_t;
```

**Required for Go**:
- Central game coordinator
- System lifecycle management
- Cross-system communication

## 📦 Required Package Installation

### C++ Dependencies (Original)
```bash
# Graphics
libgl1-mesa-dev
libglu1-mesa-dev

# Audio  
libopenal-dev
libvorbis-dev

# Video
libavformat-dev
libavcodec-dev
libavutil-dev

# UI
libx11-dev
```

### Go Dependencies (Needed)
```bash
go get github.com/hajimehoshi/ebiten/v2        # ✅ Already installed
go get github.com/hajimehoshi/oto/v2           # ❌ Audio output
go get github.com/jfreymuth/oggvorbis          # ❌ OGG decoder  
go get github.com/gen2brain/raylib-go/raylib   # ❌ Alternative audio
# Video: Need to research Go FFmpeg bindings
```

## 🎯 Implementation Priority

### Phase 1: Critical Systems (Audio + Integration)
1. **Audio System Implementation**
   - Basic audio playback
   - OGG Vorbis support
   - BGM/SE/Voice channels

2. **System Integration**
   - Central game coordinator
   - Cross-system event handling
   - Resource lifecycle management

### Phase 2: Enhanced Features
1. **Script File Parser**
   - Parse actual School Days script files
   - Event factory system

2. **Video System**
   - Basic video playback
   - Texture integration

### Phase 3: Polish & Optimization
1. **Advanced Graphics**
   - Video texture rendering
   - Advanced blending modes

2. **Performance Optimization**
   - Memory management
   - Resource caching

## 🔗 Integration Points

### Current Working Integrations
- ✅ **Graphics ↔ Menu**: Menu states trigger graphics changes
- ✅ **Settings ↔ Graphics**: Screen resolution, etc.
- ✅ **Filesystem ↔ Graphics**: Texture loading
- ✅ **Script ↔ Graphics**: Fade effects (basic)

### Missing Critical Integrations
- ❌ **Script ↔ Audio**: Script events should trigger audio
- ❌ **Menu ↔ Audio**: Menu interactions should play sounds  
- ❌ **Script ↔ Video**: Script events should control video playback
- ❌ **Audio ↔ Settings**: Volume controls
- ❌ **Global ↔ All**: Central coordinator managing all systems

## 📁 Expected File Structure (C++ Reference)

### Configuration Files
```
game.ini           # ✅ Main game config  
Settings.ini       # ✅ User settings
Ini/
├── DX9Graphic.ini    # Graphics settings
├── DX8Sound.ini      # Audio settings  
├── FILMEngine.ini    # Video settings
├── StartScript.ini   # Initial script
├── DebugInfo.ini     # Debug config
└── EndList.ini       # Ending definitions
```

### Asset Structure
```
System/
├── Screen/           # ✅ Black/White textures
├── Title/           # ✅ Title screen assets
└── Menu/            # ✅ Menu backgrounds

Script/              # ❌ MISSING: Actual script files
├── 00/, 01/, 02/    # Story chapters
├── RUSSIAN/         # Localization
└── *.tab files      # Timeline scripts

BGM/                 # ❌ MISSING: Background music
Voice00-05/          # ❌ MISSING: Voice files  
Se00-05/             # ❌ MISSING: Sound effects
Movie00-05/          # ❌ MISSING: Video files
Event00-05/          # ❌ MISSING: Event data
```

## 🎮 Next Steps Recommendation

1. **Implement Audio System** - This is blocking script-to-audio integration
2. **Create System Coordinator** - Central hub for all systems like C++ `global`
3. **Add Script File Parser** - To load actual School Days content
4. **Test with Real Assets** - Use actual School Days GPK files
5. **Add Video Support** - For complete visual novel experience

The Go engine has excellent foundations but needs the audio system and system integration to be functionally equivalent to the C++ version.
