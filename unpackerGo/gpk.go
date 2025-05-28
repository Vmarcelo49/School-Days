package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"
)

// GPK constants and structures
const (
	GPKTailerIdent0 = "STKFile0PIDX"
	GPKTailerIdent1 = "STKFile0PACKFILE"
)

var cipherCode = [16]byte{
	0x82, 0xEE, 0x1D, 0xB3,
	0x57, 0xE9, 0x2C, 0xC2,
	0x2F, 0x54, 0x7B, 0x10,
	0x4C, 0x9A, 0x75, 0x49,
}

// GPKSignature represents the GPK file signature
type GPKSignature struct {
	Sig0       [12]byte
	PidxLength uint32
	Sig1       [16]byte
}

// GPKEntryHeader represents an entry header in the GPK file
type GPKEntryHeader struct {
	SubVersion   uint16
	Version      uint16
	Zero         uint16
	Offset       uint32
	ComprLen     uint32
	Dflt         [4]byte
	UncomprLen   uint32
	ComprHeadLen byte
}

// GPKEntry represents a file entry in the GPK package
type GPKEntry struct {
	Name   string
	Header GPKEntryHeader
}

// GPK represents a GPK package file
type GPK struct {
	entries  []GPKEntry
	fileName string
}

// NewGPK creates a new GPK instance
func NewGPK() *GPK {
	return &GPK{
		entries: make([]GPKEntry, 0),
	}
}

// Load loads a GPK file and parses its contents
func (g *GPK) Load(fileName string) error {
	g.fileName = fileName

	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}
	fileSize := stat.Size()

	// Read signature from the end of file
	var signature GPKSignature
	_, err = file.Seek(fileSize-int64(binary.Size(signature)), 0)
	if err != nil {
		return fmt.Errorf("failed to seek to signature: %w", err)
	}

	err = binary.Read(file, binary.LittleEndian, &signature)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}

	// Verify signature
	if string(signature.Sig0[:len(GPKTailerIdent0)]) != GPKTailerIdent0 ||
		string(signature.Sig1[:len(GPKTailerIdent1)]) != GPKTailerIdent1 {
		return fmt.Errorf("invalid GPK signature")
	}

	// Read compressed PIDX data
	pidxOffset := fileSize - int64(binary.Size(signature)) - int64(signature.PidxLength)
	_, err = file.Seek(pidxOffset, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to PIDX data: %w", err)
	}

	compressedData := make([]byte, signature.PidxLength)
	_, err = file.Read(compressedData)
	if err != nil {
		return fmt.Errorf("failed to read compressed data: %w", err)
	}

	// Decrypt data
	for i := 0; i < len(compressedData); i++ {
		compressedData[i] ^= cipherCode[i%16]
	}

	// Decompress data (Qt uses big-endian size prefix, zlib expects big-endian)
	// Qt's qUncompress format: 4 bytes size (big-endian) + zlib data
	if len(compressedData) < 4 {
		return fmt.Errorf("compressed data too short")
	}

	// Skip the first 4 bytes (Qt's size prefix) and decompress
	zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData[4:]))
	if err != nil {
		return fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zlibReader.Close()

	uncompressedData, err := io.ReadAll(zlibReader)
	if err != nil {
		return fmt.Errorf("failed to decompress data: %w", err)
	}

	// Parse entries
	err = g.parseEntries(uncompressedData)
	if err != nil {
		return fmt.Errorf("failed to parse entries: %w", err)
	}

	return nil
}

// parseEntries parses the uncompressed PIDX data to extract file entries
func (g *GPK) parseEntries(data []byte) error {
	reader := bytes.NewReader(data)

	for reader.Len() > 0 {
		var entry GPKEntry

		// Read filename length
		var filenameLen uint16
		err := binary.Read(reader, binary.LittleEndian, &filenameLen)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read filename length: %w", err)
		}

		// Read filename (UTF-16LE)
		filenameBytes := make([]byte, filenameLen*2)
		_, err = reader.Read(filenameBytes)
		if err != nil {
			return fmt.Errorf("failed to read filename: %w", err)
		}

		// Convert UTF-16LE to string
		utf16Data := make([]uint16, filenameLen)
		for i := 0; i < int(filenameLen); i++ {
			utf16Data[i] = binary.LittleEndian.Uint16(filenameBytes[i*2 : i*2+2])
		}
		entry.Name = string(utf16.Decode(utf16Data))

		// Read entry header
		err = binary.Read(reader, binary.LittleEndian, &entry.Header)
		if err != nil {
			return fmt.Errorf("failed to read entry header: %w", err)
		}

		g.entries = append(g.entries, entry)
	}

	return nil
}

// GetName returns the package name without path and extension
func (g *GPK) GetName() string {
	base := filepath.Base(g.fileName)
	return strings.TrimSuffix(base, ".GPK")
}

// UnpackAll unpacks all files in the GPK to the specified directory
func (g *GPK) UnpackAll(outputDir string) error {
	file, err := os.Open(g.fileName)
	if err != nil {
		return fmt.Errorf("failed to open GPK file: %w", err)
	}
	defer file.Close()

	for _, entry := range g.entries {
		// Create output directory
		outputPath := filepath.Join(outputDir, entry.Name)
		outputDirPath := filepath.Dir(outputPath)

		err := os.MkdirAll(outputDirPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDirPath, err)
		}

		// Read file data
		_, err = file.Seek(int64(entry.Header.Offset), 0)
		if err != nil {
			return fmt.Errorf("failed to seek to entry %s: %w", entry.Name, err)
		}

		fileData := make([]byte, entry.Header.ComprLen)
		_, err = file.Read(fileData)
		if err != nil {
			return fmt.Errorf("failed to read entry %s: %w", entry.Name, err)
		}

		// Write file
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
		}

		_, err = outFile.Write(fileData)
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}

		fmt.Printf("Extracted: %s\n", entry.Name)
	}

	return nil
}

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

// matchPatternExact performs pattern matching that mimics Qt's QRegExp.exactMatch() behavior
// This function supports full regular expressions like the original Qt implementation
// Currently unused but available for compatibility with original behavior
func matchPatternExact(pattern, name string) bool {
	// Handle empty pattern
	if pattern == "" {
		return name == ""
	}

	// Handle simple wildcard case for performance
	if pattern == "*" {
		return true
	}

	// Convert Qt-style wildcards to regex if needed
	// Qt QRegExp supports both regex and wildcard patterns
	regexPattern := convertQtPatternToRegex(pattern)

	// Compile regex with case-insensitive flag (Qt::CaseInsensitive)
	regex, err := regexp.Compile("(?i)^" + regexPattern + "$")
	if err != nil {
		// If regex compilation fails, fall back to simple string matching
		return strings.EqualFold(pattern, name)
	}

	// Use MatchString for exact match behavior (entire string must match)
	return regex.MatchString(name)
}

// convertQtPatternToRegex converts Qt-style patterns to Go regex
// Qt QRegExp supports both wildcard and regex syntax
func convertQtPatternToRegex(pattern string) string {
	// If pattern already looks like a regex (contains regex metacharacters), use as-is
	regexMetaChars := []string{"[", "]", "(", ")", "{", "}", "^", "$", "|", "+"}
	hasRegexChars := false
	for _, char := range regexMetaChars {
		if strings.Contains(pattern, char) {
			hasRegexChars = true
			break
		}
	}

	if hasRegexChars {
		// Pattern likely contains regex syntax, use it directly
		return pattern
	}

	// Convert wildcard pattern to regex
	// Escape special regex characters first
	escaped := regexp.QuoteMeta(pattern)

	// Convert escaped wildcards back to regex equivalents
	// \* becomes .* (any characters)
	// \? becomes .  (any single character)
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")
	escaped = strings.ReplaceAll(escaped, "\\?", ".")

	return escaped
}
