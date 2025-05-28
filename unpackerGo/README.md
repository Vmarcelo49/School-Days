# School Days GPK Unpacker (Go Port)

This is a Go port of the School Days GPK unpacker, originally written in C++ with Qt.

## Overview

The School Days game stores its assets in GPK (Game Package) files. This unpacker can extract all files from these packages.

## Features

- Extracts all GPK files from the game's `packs` directory
- Handles encrypted and compressed file entries
- Preserves directory structure
- Cross-platform (Windows, Linux, macOS)
- No external GUI dependencies

## Usage

### Building

```bash
cd unpackerGo
go mod tidy
go build
```

### Running

```bash
# Use default path (hardcoded)
./school-days-unpacker

# Or specify custom game root path
./school-days-unpacker "/path/to/school-days"
```

### Example

```bash
# Extract all GPK files from the game directory
./school-days-unpacker "C:\Games\SchoolDays"
```

## File Structure

The unpacker will:
1. Look for GPK files in the `packs` subdirectory of the game root
2. Create output directories named after each GPK file
3. Extract all files maintaining their internal directory structure

## GPK File Format

GPK files use a custom format with:
- Encrypted and compressed file index (PIDX)
- XOR cipher for decryption
- Zlib compression for the file index
- UTF-16LE encoded filenames

## Dependencies

- Go 1.21 or later
- No external dependencies beyond the Go standard library

## Differences from Original

- Pure Go implementation (no Qt dependency)
- Command-line only (no GUI)
- Simplified error handling
- Cross-platform file path handling

## License

This port maintains compatibility with the original project's license.
