package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Open opens a specific file from the GPK package
func (g *GPK) Open(filename string) (*GPKFile, error) {
	for _, entry := range g.entries {
		if strings.EqualFold(entry.Name, filename) {
			return NewGPKFileFromPackage(&entry.Header, g.fileName)
		}
	}
	return nil, fmt.Errorf("file not found in package: %s", filename)
}

// List returns files matching a pattern (simple wildcard matching)
func (g *GPK) List(pattern string) []string {
	var result []string

	for _, entry := range g.entries {
		if matchPattern(pattern, entry.Name) {
			result = append(result, entry.Name)
		}
	}

	return result
}

// GetName returns the base name of the GPK file without extension
func (g *GPK) GetName() string {
	// Extract base filename without path and extension
	filename := filepath.Base(g.fileName)
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		filename = filename[:idx]
	}
	return filename
}

// GetEntries returns all entries in the GPK
func (g *GPK) GetEntries() []GPKEntry {
	return g.entries
}

// matchPattern performs simple wildcard matching (* and ?)
func matchPattern(pattern, name string) bool {
	// Simple case - exact match or empty pattern
	if pattern == "" || pattern == "*" {
		return true
	}

	// Convert to uppercase for case-insensitive matching
	pattern = strings.ToUpper(pattern)
	name = strings.ToUpper(name)

	// Simple wildcard matching
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			// Pattern like "*.ext" or "prefix*"
			if parts[0] == "" {
				// Suffix match
				return strings.HasSuffix(name, parts[1])
			} else if parts[1] == "" {
				// Prefix match
				return strings.HasPrefix(name, parts[0])
			} else {
				// Contains both prefix and suffix
				return strings.HasPrefix(name, parts[0]) && strings.HasSuffix(name, parts[1])
			}
		}
	}

	// Exact match (case insensitive)
	return pattern == name
}
