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
	ComprHeadLen uint8
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

	// Decompress data - Qt's qUncompress format requires special handling
	if len(compressedData) < 4 {
		return fmt.Errorf("compressed data too short")
	}

	// Qt's qUncompress format: modify the first 4 bytes to make it compatible with zlib
	originalSize := make([]byte, 4)
	copy(originalSize, compressedData[:4])
	compressedData[0] = 0
	compressedData[1] = 0
	compressedData[2] = originalSize[1]
	compressedData[3] = originalSize[0]

	// Decompress starting from byte 4
	zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData[4:]))
	if err != nil {
		return fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zlibReader.Close()

	uncompressedData, err := io.ReadAll(zlibReader)
	if err != nil {
		return fmt.Errorf("failed to decompress data: %w", err)
	}

	// Parse entries using the fixed header size from C++ struct
	err = g.parseEntries(uncompressedData)
	if err != nil {
		return fmt.Errorf("failed to parse entries: %w", err)
	}

	return nil
}

// parseEntries parses the uncompressed PIDX data to extract file entries
func (g *GPK) parseEntries(data []byte) error {
	offset := 0
	dataLen := len(data)
	headerSize := 23 // Fixed size based on C++ struct

	for offset < dataLen {
		// Check if we have at least 2 bytes for filename length
		if offset+2 > dataLen {
			break
		}

		// Read filename length
		filenameLen := binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2

		// Sanity check for filename length
		if filenameLen == 0 {
			break
		}
		if filenameLen > 1024 {
			return fmt.Errorf("invalid filename length: %d at offset %d", filenameLen, offset-2)
		}

		// Check if we have enough data remaining for filename
		if offset+int(filenameLen)*2 > dataLen {
			return fmt.Errorf("not enough data for filename: need %d bytes, have %d", filenameLen*2, dataLen-offset)
		}

		// Read filename (UTF-16LE)
		filenameBytes := data[offset : offset+int(filenameLen)*2]
		offset += int(filenameLen) * 2

		// Convert UTF-16LE to string
		utf16Data := make([]uint16, filenameLen)
		for i := range int(filenameLen) {
			utf16Data[i] = binary.LittleEndian.Uint16(filenameBytes[i*2 : i*2+2])
		}
		filename := string(utf16.Decode(utf16Data))

		// Check if we have enough data for header
		if offset+headerSize > dataLen {
			return fmt.Errorf("not enough data for header: need %d bytes, have %d", headerSize, dataLen-offset)
		}

		// Read header using fixed 23-byte size
		headerBytes := data[offset : offset+headerSize]
		offset += headerSize

		// Parse the header manually
		var entry GPKEntry
		entry.Name = filename

		headerReader := bytes.NewReader(headerBytes)
		err := binary.Read(headerReader, binary.LittleEndian, &entry.Header.SubVersion)
		if err != nil {
			return fmt.Errorf("failed to read SubVersion: %w", err)
		}
		err = binary.Read(headerReader, binary.LittleEndian, &entry.Header.Version)
		if err != nil {
			return fmt.Errorf("failed to read Version: %w", err)
		}
		err = binary.Read(headerReader, binary.LittleEndian, &entry.Header.Zero)
		if err != nil {
			return fmt.Errorf("failed to read Zero: %w", err)
		}
		err = binary.Read(headerReader, binary.LittleEndian, &entry.Header.Offset)
		if err != nil {
			return fmt.Errorf("failed to read Offset: %w", err)
		}
		err = binary.Read(headerReader, binary.LittleEndian, &entry.Header.ComprLen)
		if err != nil {
			return fmt.Errorf("failed to read ComprLen: %w", err)
		}
		_, err = headerReader.Read(entry.Header.Dflt[:])
		if err != nil {
			return fmt.Errorf("failed to read Dflt: %w", err)
		}
		err = binary.Read(headerReader, binary.LittleEndian, &entry.Header.UncomprLen)
		if err != nil {
			return fmt.Errorf("failed to read UncomprLen: %w", err)
		}
		err = binary.Read(headerReader, binary.LittleEndian, &entry.Header.ComprHeadLen)
		if err != nil {
			return fmt.Errorf("failed to read ComprHeadLen: %w", err)
		}

		// Add the successfully parsed entry to our collection
		g.entries = append(g.entries, entry)

		// Check if this might be the end - look for patterns that suggest no more entries
		if offset < dataLen-2 {
			nextPotentialLen := binary.LittleEndian.Uint16(data[offset : offset+2])
			if nextPotentialLen == 0 {
				break
			}
			if nextPotentialLen > 1024 {
				// Look for entry separator patterns: "OggS", "Ogg" + length, "Og" + length
				found := false
				for tryOffset := offset; tryOffset < offset+10 && tryOffset < dataLen-4; tryOffset++ {
					// Check for "OggS" pattern
					if tryOffset+4 <= dataLen &&
						data[tryOffset] == 79 && data[tryOffset+1] == 103 &&
						data[tryOffset+2] == 103 && data[tryOffset+3] == 83 { // "OggS"

						// Look for next filename length after OggS + padding
						for nextOffset := tryOffset + 4; nextOffset < tryOffset+10 && nextOffset < dataLen-2; nextOffset++ {
							tryLen := binary.LittleEndian.Uint16(data[nextOffset : nextOffset+2])
							if tryLen > 0 && tryLen <= 100 && g.isValidFilename(data, nextOffset, tryLen) {
								offset = nextOffset
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					// Check for "Ogg" + length pattern
					if !found && tryOffset+4 <= dataLen &&
						data[tryOffset] == 79 && data[tryOffset+1] == 103 &&
						data[tryOffset+2] == 103 { // "Ogg"

						potentialLen := data[tryOffset+3]
						if potentialLen > 0 && potentialLen <= 100 {
							nextOffset := tryOffset + 3
							if g.isValidFilename(data, nextOffset, uint16(potentialLen)) {
								offset = nextOffset
								found = true
								break
							}
						}
					}
				}
				// Check for "Og" + length pattern
				if !found {
					for tryOffset := offset; tryOffset < offset+10 && tryOffset < dataLen-3; tryOffset++ {
						if data[tryOffset] == 79 && data[tryOffset+1] == 103 { // "Og"
							potentialLen := data[tryOffset+2]
							if potentialLen > 0 && potentialLen <= 100 {
								nextOffset := tryOffset + 2
								if g.isValidFilename(data, nextOffset, uint16(potentialLen)) {
									offset = nextOffset
									found = true
									break
								}
							}
						}
					}
				}
				if !found {
					break
				} else {
					continue // Try parsing from this new offset
				}
			}
		}
	}

	return nil
}

// isValidFilename checks if the data at the given offset contains a valid UTF-16 filename
func (g *GPK) isValidFilename(data []byte, offset int, filenameLen uint16) bool {
	if offset+2+int(filenameLen)*2 >= len(data) {
		return false
	}

	nameBytes := data[offset+2 : offset+2+int(filenameLen)*2]
	utf16Data := make([]uint16, filenameLen)

	for i := 0; i < int(filenameLen); i++ {
		utf16Data[i] = binary.LittleEndian.Uint16(nameBytes[i*2 : i*2+2])
		// Check if this looks like a reasonable character
		if utf16Data[i] == 0 || utf16Data[i] > 0xFFFF {
			return false
		}
	}

	possibleName := string(utf16.Decode(utf16Data))
	// Check if the name looks like a reasonable filename
	return strings.Contains(possibleName, "/") && strings.Contains(possibleName, ".")
}

// GetName returns the package name without path and extension
func (g *GPK) GetName() string {
	base := filepath.Base(g.fileName)
	return strings.TrimSuffix(base, ".GPK")
}

// GetEntries returns all entries in the GPK file
func (g *GPK) GetEntries() []GPKEntry {
	return g.entries
}

// UnpackAll unpacks all files in the GPK to the specified directory
func (g *GPK) UnpackAll(outputDir string) error {
	file, err := os.Open(g.fileName)
	if err != nil {
		return fmt.Errorf("failed to open GPK file: %w", err)
	}
	defer file.Close()

	for i, entry := range g.entries {
		fmt.Printf("Extracting %d/%d: %s\n", i+1, len(g.entries), entry.Name)

		// Create output directory
		outputPath := filepath.Join(outputDir, entry.Name)
		outputDirPath := filepath.Dir(outputPath)

		err := os.MkdirAll(outputDirPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDirPath, err)
		} // Read file data
		_, err = file.Seek(int64(entry.Header.Offset), 0)
		if err != nil {
			return fmt.Errorf("failed to seek to entry %s: %w", entry.Name, err)
		}

		// Read all data including header
		allData := make([]byte, entry.Header.ComprLen)
		_, err = file.Read(allData)
		if err != nil {
			return fmt.Errorf("failed to read entry %s: %w", entry.Name, err)
		}

		// Find the actual OGG data start by searching for "OggS" pattern
		var fileData []byte
		oggOffset := -1

		// Search for "OggS" pattern in the data
		for i := 0; i < len(allData)-3; i++ {
			if allData[i] == 'O' && allData[i+1] == 'g' && allData[i+2] == 'g' && allData[i+3] == 'S' {
				oggOffset = i
				break
			}
		}

		if oggOffset >= 0 {
			// Found OGG data, extract from that position
			fileData = allData[oggOffset:]
			fmt.Printf("  Found OGG data at offset %d, skipping %d header bytes for %s\n", oggOffset, oggOffset, entry.Name)
		} else {
			// No OGG pattern found, try skipping compression header as fallback
			if entry.Header.ComprHeadLen > 0 && int(entry.Header.ComprHeadLen) < len(allData) {
				fileData = allData[entry.Header.ComprHeadLen:]
				fmt.Printf("  No OGG pattern found, skipping %d compression header bytes for %s\n", entry.Header.ComprHeadLen, entry.Name)
			} else {
				// Use all data as-is
				fileData = allData
				fmt.Printf("  No header processing for %s\n", entry.Name)
			}
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
	regexPattern := convertQtPatternToRegex(pattern)

	// Compile regex with case-insensitive flag
	regex, err := regexp.Compile("(?i)^" + regexPattern + "$")
	if err != nil {
		// If regex compilation fails, fall back to simple string matching
		return strings.EqualFold(pattern, name)
	}

	return regex.MatchString(name)
}

// convertQtPatternToRegex converts Qt-style patterns to Go regex
func convertQtPatternToRegex(pattern string) string {
	// If pattern already looks like a regex, use as-is
	regexMetaChars := []string{"[", "]", "(", ")", "{", "}", "^", "$", "|", "+"}
	hasRegexChars := false
	for _, char := range regexMetaChars {
		if strings.Contains(pattern, char) {
			hasRegexChars = true
			break
		}
	}

	if hasRegexChars {
		return pattern
	}

	// Convert wildcard pattern to regex
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")
	escaped = strings.ReplaceAll(escaped, "\\?", ".")

	return escaped
}
