AI generated readme based on the original project:

# School Days (SD) - Visual Novel Game Engine

A C++ implementation of a visual novel game engine designed to run School Days visual novel games. This project includes a complete game engine, Qt-based editor, and utilities for handling game assets.

## Project Overview

This is an open-source implementation of a School Days visual novel game engine written in C++. The project aims to recreate the functionality needed to play School Days games using modern cross-platform technologies including OpenGL, OpenAL, and Qt.

**Important**: This project contains **three separate executable components**, each serving a different purpose:

1. **Game Engine** (`cpp_incorrect_ver/`) - The main School Days game runtime
2. **Script Editor** (`sd_editor/`) - Qt-based visual script editor and viewer  
3. **GPK Unpacker** (`unpacker/`) - Utility to extract game asset packages

Each component has its own `main.cpp` file and build system.

### Key Features

- **Visual Novel Engine**: Complete game engine supporting script execution, multimedia playback, and user interaction
- **GPK Package System**: Support for the proprietary GPK package format used by School Days games
- **Script System**: JRS/ORS script format support for game scenarios and dialogue
- **Qt-based Editor**: Visual editor for viewing and editing game scripts
- **Cross-platform**: Designed to work on Linux with X11, OpenGL, OpenAL, and PThread support
- **Asset Management**: Comprehensive file system for handling game assets (images, audio, video, scripts)

## Project Structure

```
SD/
├── cpp_incorrect_ver/     # Main game engine implementation
│   ├── main.cpp          # Application entry point
│   ├── game.cpp          # Core game logic
│   ├── script_engine.cpp # Script execution engine
│   ├── gpk.cpp           # GPK package format handler
│   └── interface.cpp     # User interface management
├── sd_editor/            # Qt-based visual script editor
│   ├── mainwindow.cpp    # Editor main window
│   ├── sd_editor.pro     # Qt project file
│   └── main.cpp          # Editor entry point
├── classes/              # Shared Qt classes
│   ├── qfilesystem.cpp   # File system abstraction
│   ├── qgpk.cpp          # Qt GPK package handler
│   ├── qscript.cpp       # Qt script engine
│   └── qgpkfile.cpp      # GPK file wrapper
├── unpacker/             # Utility for extracting GPK packages
├── game/                 # Game configuration and assets
│   └── game.json         # Game folder structure definition
└── disasm/               # Disassembly and reverse engineering files
```

## Architecture

### Core Components

1. **Game Engine** (`cpp_incorrect_ver/`):
   - Main game loop and initialization
   - Graphics rendering with OpenGL
   - Audio playback with OpenAL
   - Input handling and user interface
   - Script execution and event management

2. **File System** (`classes/qfilesystem.cpp`):
   - Virtual file system supporting both regular files and GPK packages
   - Automatic GPK package mounting from `packs/` directory
   - File path normalization and case-insensitive lookups

3. **Script Engine** (`script_engine.cpp`, `qscript.cpp`):
   - Event-based script execution system
   - Support for multimedia events (background images, videos, audio)
   - Text display and user interaction handling
   - Timeline-based event scheduling

4. **GPK Package System**:
   - Custom package format used by School Days games
   - Compressed file storage with encryption
   - Support for various asset types (images, audio, video, scripts)

### File Formats

#### GPK Package Format
- **Purpose**: Container format for game assets
- **Features**: 
  - Compressed data storage using zlib
  - Encrypted file index with custom cipher
  - Support for multiple asset types
  - Directory structure preservation

#### ORS/JRS Script Format
- **ORS**: Original script format used by School Days
- **JRS**: JSON-based script format for easier editing
- **Actions Supported**:
  - `CreateBG`: Background image display
  - `PlayMovie`: Video playback
  - `PlayBgm`/`PlaySe`: Audio playback
  - `PlayVoice`: Character voice with lip sync
  - `PrintText`: Dialog display
  - `BlackFade`/`WhiteFade`: Screen transitions
  - `SetSELECT`: Choice selection menus

### Asset Organization

The game organizes assets into specific folders as defined in `game.json`:
- `ini/`: Configuration files
- `system/`: System resources
- `Script/`: Game scripts and scenarios
- `BGM/`: Background music
- `Event00-05/`: Event-specific assets
- `Movie00-05/`: Video files
- `Se00-05/`: Sound effects
- `Voice00-05/`: Character voice files
- `Commentary/`: Commentary tracks

## Entry Points and Compilation

This project contains **three separate executable components**, each with its own `main.cpp` file:

### 1. Main Game Engine (`cpp_incorrect_ver/main.cpp`)
- **Purpose**: The actual School Days game runtime
- **Entry Point**: `cpp_incorrect_ver/main.cpp`
- **Build System**: Code::Blocks project files (.cbp)
- **Target Platform**: Windows (`SD_win.cbp`) and Linux (`SD_unix.cbp`)

### 2. Qt Script Editor (`sd_editor/main.cpp`)
- **Purpose**: Visual editor for viewing and editing game scripts
- **Entry Point**: `sd_editor/main.cpp`
- **Build System**: Qt qmake project file (`sd_editor.pro`)
- **GUI**: Qt-based desktop application

### 3. GPK Unpacker Utility (`unpacker/main.cpp`)
- **Purpose**: Command-line tool to extract GPK package files
- **Entry Point**: `unpacker/main.cpp`
- **Build System**: Qt qmake project file (`unpacker.pro`)
- **Type**: Console application

## Building and Installation

### Prerequisites

#### For Game Engine:
- **C++ Compiler**: GCC or MinGW (Windows)
- **Libraries**:
  - OpenGL (`opengl32`, `glu32`)
  - OpenAL (`OpenAL32`)
  - zlib compression library
  - libpng for image handling
  - FFmpeg libraries (`avformat`, `avcodec`, `avutil`)
  - Vorbis audio libraries (`vorbisfile`, `vorbis`, `ogg`)

#### For Qt Components (Editor & Unpacker):
- **Qt Framework**: Qt 5.x with modules:
  - Core, GUI, Widgets
  - Multimedia (for editor)
- **C++ Compiler**: Compatible with Qt (GCC, MSVC, Clang)

### Building the Game Engine

The main game engine uses Code::Blocks project files:

#### Windows (using Code::Blocks):
```powershell
# Open SD_win.cbp in Code::Blocks IDE
# Or use command line if cbp2make is available:
cd cpp_incorrect_ver
cbp2make -in SD_win.cbp
make
```

#### Windows (Manual compilation):
```powershell
cd cpp_incorrect_ver
g++ -std=c++11 -Wall -fexceptions -U__STRICT_ANSI__ ^
    -IC:/MinGW/include ^
    *.cpp ^
    -lz -lpng -lopengl32 -lglu32 -lOpenAL32 -lwinmm ^
    -lvorbisfile -lvorbis -logg -lavformat -lavcodec -lavutil ^
    -mwindows ^
    -o bin/Release/SD.exe
```

#### Linux:
```bash
cd cpp_incorrect_ver
# Open SD_unix.cbp in Code::Blocks or compile manually:
g++ -std=c++11 -Wall *.cpp -lz -lpng -lGL -lGLU -lopenal -pthread \
    -lvorbisfile -lvorbis -logg -lavformat -lavcodec -lavutil \
    -o bin/Release/SD
```

### Building the Qt Editor

```powershell
cd sd_editor
qmake sd_editor.pro
# On Windows with MSVC:
nmake
# On Windows with MinGW or Linux:
make
```

### Building the Unpacker Utility

```powershell
cd unpacker
qmake unpacker.pro
# On Windows with MSVC:
nmake
# On Windows with MinGW or Linux:
make
```

### Alternative: Using Qt Creator

Both Qt components can be easily built using Qt Creator IDE:
1. Open `sd_editor/sd_editor.pro` or `unpacker/unpacker.pro`
2. Configure the project for your Qt kit
3. Build and run

## Usage

### Running the Game

1. Place game assets in the appropriate directory structure
2. Ensure GPK packages are in the `packs/` subdirectory
3. Configure `game.ini` with appropriate settings
4. Run the game executable

### Using the Editor

1. Launch the SD Editor application
2. Set the game folder containing GPK packages
3. Browse and select scripts from the script tree
4. View script execution in the engine widget
5. Export scripts to text format for translation/editing

### Extracting Game Assets

Use the unpacker utility to extract GPK packages:

```bash
./unpacker
# Configure the game path in the source code or pass as argument
```

## Configuration

### Game Settings (`game.ini`)

The game engine loads configuration from various INI files:
- Graphics settings (resolution, fullscreen mode)
- Audio settings (sound devices, volume levels)
- File system settings (asset extensions, package locations)
- Debug settings

### Supported Asset Formats

- **Images**: PNG format for backgrounds and character sprites
- **Audio**: Support for various formats through OpenAL
- **Video**: WMV format for movie playback
- **Scripts**: ORS (original) and JRS (JSON) formats

## Development Notes

### Current Status

This appears to be a reverse-engineering project of the School Days game engine. The code includes:
- Partial implementation of the game engine
- Working GPK package format support
- Functional Qt-based editor
- Script parsing and execution framework

### Known Limitations

- Some features may be incomplete (marked as "incorrect_ver")
- Platform-specific code primarily targets Linux
- May require original game assets to function properly

## License

This project uses **dual licensing** with different components under different GPL versions:

- **Main Project**: Licensed under **GNU General Public License v2.0** (see root `LICENSE` file)
- **Game Engine** (`cpp_incorrect_ver/`): Licensed under **GNU General Public License v3.0** (see `cpp_incorrect_ver/LICENSE`)

### What this means for usage:

#### ✅ **You CAN:**
- **Use the software** for any purpose (personal, educational, commercial)
- **Study and modify** the source code
- **Distribute copies** of the original software
- **Distribute your modifications** (under the same GPL license)
- **Run the software** without restrictions

#### ⚠️ **You MUST:**
- **Provide source code** when distributing binaries
- **Keep the same license** (GPL v2/v3) for any modifications
- **Include copyright notices** and license text
- **Document any changes** you make to the code
- **Make your modifications available** under GPL if you distribute them

#### ❌ **You CANNOT:**
- **Incorporate this code into proprietary software** without making it GPL
- **Remove or modify** the license notices
- **Use this in closed-source projects** (due to copyleft nature)
- **Sublicense** under different terms

### GPL Copyleft Effect

Both GPL v2 and GPL v3 are **copyleft licenses**, meaning any derivative work must also be released under a compatible GPL license. This ensures the code remains free and open source.

### For Commercial Use

You can use this code commercially, but any commercial distribution must:
1. Include full source code
2. Maintain GPL licensing
3. Allow users to modify and redistribute
4. Provide build instructions

## Disclaimer

This is a reverse-engineering project for educational and preservation purposes. Users must own legitimate copies of School Days games to use this engine with game assets. The original School Days game and assets are copyrighted by their respective owners.

**Important**: This project only covers the engine implementation. Game assets, artwork, music, and script content remain under their original copyright and are not covered by the GPL license.

## Contributing

This project appears to be focused on reverse engineering and reimplementation. Contributions should respect intellectual property rights and focus on clean-room implementations based on publicly available information.

## Technical Details

### GPK Package Internals

The GPK format uses:
- Custom encryption with a 16-byte cipher key
- zlib compression for the file index
- Little-endian byte order
- UTF-16 filenames converted to UTF-8

### Script Engine Events

The script engine supports timeline-based events:
- Events have start and end times
- Support for parallel event execution
- Fade effects with mathematical easing
- Media synchronization capabilities

### Qt Integration

The Qt components provide:
- Cross-platform file system abstraction
- Multimedia playback integration
- Visual script editing capabilities
- Asset preview and management tools
