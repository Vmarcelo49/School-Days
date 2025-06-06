package settings

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"school-days-engine/internal/filesystem"
)

// INIManager handles INI-based configuration like the original C++ Settings class
type INIManager struct {
	settings map[string]string // All values stored as strings initially
	fs       *filesystem.Manager
}

// NewINIManager creates a new INI-based settings manager (matches C++ Settings)
func NewINIManager(fs *filesystem.Manager) *INIManager {
	return &INIManager{
		settings: make(map[string]string),
		fs:       fs,
	}
}

// Load loads an INI file and merges it with existing settings (matches C++ Settings::load)
func (m *INIManager) Load(filename string) error {
	log.Printf("Loading INI file: %s", filename)

	reader, err := m.fs.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open INI file %s: %v", filename, err)
	}
	defer reader.Close()

	return m.parseINI(reader)
}

// GetString returns a string value (matches C++ Settings::get_string)
func (m *INIManager) GetString(key string) string {
	if value, exists := m.settings[key]; exists {
		// Remove quotes if present (C++ format uses quotes)
		return strings.Trim(value, `"`)
	}
	return ""
}

// GetInt returns an integer value (matches C++ Settings::get_int)
func (m *INIManager) GetInt(key string) int {
	value := m.GetString(key)
	if value == "" {
		return -1 // C++ returns -1 for missing values
	}

	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	return -1
}

// GetFloat returns a float value (matches C++ Settings::get_float)
func (m *INIManager) GetFloat(key string) float64 {
	value := m.GetString(key)
	if value == "" {
		return -1.0 // C++ returns -1.0 for missing values
	}

	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	return -1.0
}

// GetBool returns a boolean value (matches C++ Settings::get_bool)
func (m *INIManager) GetBool(key string) bool {
	value := m.GetString(key)
	if value == "" {
		return false // C++ returns false for missing values
	}
	
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal != 0 // C++ treats non-zero as true
	}
	return false
}

// SetInt sets an integer value (matches C++ Settings::set_int)
func (m *INIManager) SetInt(key string, value int) {
	m.settings[key] = strconv.Itoa(value)
	log.Printf("Settings: set %s = %d", key, value)
}

// parseINI parses an INI file from a reader (matches C++ Parser behavior)
func (m *INIManager) parseINI(reader io.ReadCloser) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse C++ format: [key]="value" or [key]=value
		if err := m.parseLine(line); err != nil {
			log.Printf("Warning: failed to parse line '%s': %v", line, err)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading INI file: %v", err)
	}

	return nil
}

// parseLine parses a single INI line in C++ format: [key]="value"
func (m *INIManager) parseLine(line string) error {
	// Check if line starts with [
	if !strings.HasPrefix(line, "[") {
		return fmt.Errorf("line does not start with [")
	}

	// Find the closing ]
	closeBracketPos := strings.Index(line, "]")
	if closeBracketPos == -1 {
		return fmt.Errorf("missing closing bracket ]")
	}

	// Extract key
	key := line[1:closeBracketPos]
	if key == "" {
		return fmt.Errorf("empty key")
	}

	// Check for = after ]
	remaining := line[closeBracketPos+1:]
	if !strings.HasPrefix(remaining, "=") {
		return fmt.Errorf("missing = after key")
	}

	// Extract value (remove leading =)
	value := remaining[1:]

	// Handle quoted values (remove quotes if present)
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) >= 2 {
		value = value[1 : len(value)-1]
	}

	// Store the key-value pair
	m.settings[key] = value

	return nil
}

// GetAllSettings returns all current settings for debugging
func (m *INIManager) GetAllSettings() map[string]string {
	result := make(map[string]string)
	for k, v := range m.settings {
		result[k] = v
	}
	return result
}
