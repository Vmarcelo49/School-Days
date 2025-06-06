package audio

import (
	"fmt"
	"path/filepath"
	"strings"
)

// IsOggFile checks if a filename represents an OGG audio file
func IsOggFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".ogg")
}

// ParseGPKPath parses a GPK audio path into package and entry components
// Format: "package.gpk::path/to/audio.ogg" or just "audio.ogg"
func ParseGPKPath(path string) (gpkFile, entryPath string, isGPK bool) {
	if strings.Contains(path, "::") {
		parts := strings.SplitN(path, "::", 2)
		return parts[0], parts[1], true
	}
	return "", path, false
}

// BuildGPKPath constructs a GPK path from components
func BuildGPKPath(gpkFile, entryPath string) string {
	return fmt.Sprintf("%s::%s", gpkFile, entryPath)
}

// ValidateAudioPath validates an audio file path
func ValidateAudioPath(path string) error {
	if path == "" {
		return fmt.Errorf("audio path cannot be empty")
	}

	gpkFile, entryPath, isGPK := ParseGPKPath(path)

	if isGPK {
		if gpkFile == "" {
			return fmt.Errorf("GPK file name cannot be empty")
		}
		if entryPath == "" {
			return fmt.Errorf("GPK entry path cannot be empty")
		}
		if !strings.HasSuffix(strings.ToLower(gpkFile), ".gpk") {
			return fmt.Errorf("GPK file must have .gpk extension")
		}
	}

	// Check if it's an OGG file (only supported format)
	targetPath := entryPath
	if !isGPK {
		targetPath = path
	}

	if !IsOggFile(targetPath) {
		return fmt.Errorf("unsupported audio format: %s (only OGG files are supported)", filepath.Ext(targetPath))
	}

	return nil
}

// GetSystemSoundName returns the name of a system sound ID
func GetSystemSoundName(id int) string {
	switch id {
	case SndTitle:
		return "Title"
	case SndSystem:
		return "System"
	case SndCancel:
		return "Cancel"
	case SndSelect:
		return "Select"
	case SndClick:
		return "Click"
	case SndUp:
		return "Up"
	case SndDown:
		return "Down"
	case SndView:
		return "View"
	case SndOpen:
		return "Open"
	default:
		return fmt.Sprintf("Unknown (%d)", id)
	}
}

// GetSystemSoundID returns the ID for a system sound name
func GetSystemSoundID(name string) (int, bool) {
	switch strings.ToLower(name) {
	case "title":
		return SndTitle, true
	case "system":
		return SndSystem, true
	case "cancel":
		return SndCancel, true
	case "select":
		return SndSelect, true
	case "click":
		return SndClick, true
	case "up":
		return SndUp, true
	case "down":
		return SndDown, true
	case "view":
		return SndView, true
	case "open":
		return SndOpen, true
	default:
		return -1, false
	}
}
