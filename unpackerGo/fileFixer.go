package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PNG signature bytes
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
var ihdrSignature = []byte{0x49, 0x48, 0x44, 0x52} // "IHDR"

// PNGFixer handles PNG file corruption repair
type PNGFixer struct {
	FilePath       string
	OriginalData   []byte
	FixedData      []byte
	CorruptionInfo map[string]any
}

// NewPNGFixer creates a new PNG fixer for the given file
func NewPNGFixer(filePath string) *PNGFixer {
	return &PNGFixer{
		FilePath:       filePath,
		CorruptionInfo: make(map[string]any),
	}
}

// ReadFile reads the PNG file data
func (pf *PNGFixer) ReadFile() error {
	data, err := os.ReadFile(pf.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	pf.OriginalData = data
	return nil
}

// AnalyzeCorruption analyzes the corruption pattern in the PNG file
func (pf *PNGFixer) AnalyzeCorruption() error {
	if len(pf.OriginalData) < 16 {
		pf.CorruptionInfo["type"] = "file_too_small"
		return fmt.Errorf("file too small to be a valid PNG")
	}

	// Check if it already has a valid PNG signature
	if bytes.HasPrefix(pf.OriginalData, pngSignature) {
		pf.CorruptionInfo["type"] = "valid_png"
		return nil
	}

	// Look for IHDR chunk which should be early in a PNG file
	ihdrPos := bytes.Index(pf.OriginalData, ihdrSignature)
	if ihdrPos == -1 {
		pf.CorruptionInfo["type"] = "no_ihdr_found"
		return fmt.Errorf("no IHDR chunk found in file")
	}

	// Analyze what's before IHDR
	prefix := pf.OriginalData[:ihdrPos]

	pf.CorruptionInfo["type"] = "missing_signature"
	pf.CorruptionInfo["ihdr_position"] = ihdrPos
	pf.CorruptionInfo["prefix_bytes"] = prefix
	pf.CorruptionInfo["missing_bytes"] = 8 - ihdrPos + 4 // PNG signature (8) + chunk length (4) before IHDR

	// Check if we can find a chunk length before IHDR
	if ihdrPos >= 4 {
		chunkLengthBytes := pf.OriginalData[ihdrPos-4 : ihdrPos]
		chunkLength := binary.BigEndian.Uint32(chunkLengthBytes)
		pf.CorruptionInfo["ihdr_chunk_length"] = chunkLength
		pf.CorruptionInfo["ihdr_length_valid"] = (chunkLength == 13) // IHDR should be 13 bytes
	}

	return nil
}

// FixPNG attempts to reconstruct the PNG by adding missing signature
func (pf *PNGFixer) FixPNG() error {
	if err := pf.AnalyzeCorruption(); err != nil {
		return err
	}

	corruptionType, exists := pf.CorruptionInfo["type"].(string)
	if !exists {
		return fmt.Errorf("could not determine corruption type")
	}

	switch corruptionType {
	case "valid_png":
		pf.FixedData = pf.OriginalData
		return nil
	case "missing_signature":
		return pf.reconstructPNG()
	default:
		return fmt.Errorf("unsupported corruption type: %s", corruptionType)
	}
}

// reconstructPNG reconstructs the PNG by adding the missing signature
func (pf *PNGFixer) reconstructPNG() error {
	ihdrPos, exists := pf.CorruptionInfo["ihdr_position"].(int)
	if !exists {
		return fmt.Errorf("IHDR position not found")
	}

	// Strategy 1: Simply prepend the full PNG signature before chunk length
	if ihdrPos >= 4 {
		chunkStart := ihdrPos - 4
		pf.FixedData = append(pngSignature, pf.OriginalData[chunkStart:]...)

		if pf.verifyFix() {
			return nil
		}
	}

	// Strategy 2: Add PNG signature and proper chunk length if needed
	if ihdrPos < 4 {
		// Missing chunk length too, add both
		chunkLength := make([]byte, 4)
		binary.BigEndian.PutUint32(chunkLength, 13) // IHDR is always 13 bytes
		pf.FixedData = append(pngSignature, chunkLength...)
		pf.FixedData = append(pf.FixedData, pf.OriginalData[ihdrPos:]...)

		if pf.verifyFix() {
			return nil
		}
	}

	// Strategy 3: Try different offsets to find the correct data start
	for offset := 0; offset < ihdrPos; offset++ {
		testData := append(pngSignature, pf.OriginalData[offset:]...)

		// Quick verification: check if first chunk after signature looks valid
		if len(testData) >= 16 {
			chunkLength := binary.BigEndian.Uint32(testData[8:12])
			chunkType := testData[12:16]

			if bytes.Equal(chunkType, ihdrSignature) && chunkLength == 13 {
				pf.FixedData = testData
				if pf.verifyFix() {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("failed to reconstruct PNG")
}

// verifyFix verifies that the fixed data has a valid PNG structure
func (pf *PNGFixer) verifyFix() bool {
	if len(pf.FixedData) < 16 {
		return false
	}

	// Check PNG signature
	if !bytes.HasPrefix(pf.FixedData, pngSignature) {
		return false
	}

	// Check first chunk is IHDR with correct length
	chunkLength := binary.BigEndian.Uint32(pf.FixedData[8:12])
	chunkType := pf.FixedData[12:16]

	return bytes.Equal(chunkType, ihdrSignature) && chunkLength == 13
}

// SaveFixedFile saves the fixed PNG file
func (pf *PNGFixer) SaveFixedFile(suffix string) error {
	if len(pf.FixedData) == 0 {
		return fmt.Errorf("no fixed data available")
	}

	// Create new filename with suffix
	dir := filepath.Dir(pf.FilePath)
	base := filepath.Base(pf.FilePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	newPath := filepath.Join(dir, name+suffix+ext)

	err := os.WriteFile(newPath, pf.FixedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to save fixed file: %w", err)
	}

	fmt.Printf("Fixed PNG saved as: %s\n", newPath)
	return nil
}

// PrintAnalysis prints the corruption analysis
func (pf *PNGFixer) PrintAnalysis() {
	fmt.Printf("PNG Analysis for: %s\n", filepath.Base(pf.FilePath))

	corruptionType, _ := pf.CorruptionInfo["type"].(string)
	fmt.Printf("  Corruption type: %s\n", corruptionType)

	if ihdrPos, exists := pf.CorruptionInfo["ihdr_position"].(int); exists {
		fmt.Printf("  IHDR position: %d\n", ihdrPos)
	}

	if prefix, exists := pf.CorruptionInfo["prefix_bytes"].([]byte); exists {
		fmt.Printf("  Prefix bytes: %x\n", prefix)
	}

	if missingBytes, exists := pf.CorruptionInfo["missing_bytes"].(int); exists {
		fmt.Printf("  Missing bytes: %d\n", missingBytes)
	}

	if ihdrValid, exists := pf.CorruptionInfo["ihdr_length_valid"].(bool); exists {
		fmt.Printf("  IHDR length valid: %t\n", ihdrValid)
	}
}

// FixAllPNGFiles fixes all PNG files in a directory
func FixAllPNGFiles(directory string) error {
	fmt.Printf("Scanning for PNG files in: %s\n", directory)

	var pngFiles []string

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToUpper(d.Name()), ".PNG") {
			pngFiles = append(pngFiles, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(pngFiles) == 0 {
		fmt.Println("No PNG files found")
		return nil
	}

	fmt.Printf("Found %d PNG files\n", len(pngFiles))

	fixedCount := 0
	errorCount := 0

	for _, filePath := range pngFiles {
		fixer := NewPNGFixer(filePath)

		if err := fixer.ReadFile(); err != nil {
			fmt.Printf("Error reading %s: %v\n", filepath.Base(filePath), err)
			errorCount++
			continue
		}

		if err := fixer.FixPNG(); err != nil {
			fmt.Printf("Error fixing %s: %v\n", filepath.Base(filePath), err)
			errorCount++
			continue
		}

		// Check if file needed fixing
		if corruptionType, exists := fixer.CorruptionInfo["type"].(string); exists && corruptionType != "valid_png" {
			fixer.PrintAnalysis()

			if err := fixer.SaveFixedFile("_fixed"); err != nil {
				fmt.Printf("Error saving fixed file %s: %v\n", filepath.Base(filePath), err)
				errorCount++
				continue
			}
			fixedCount++
		}
	}

	fmt.Printf("\nResults: %d files fixed, %d errors\n", fixedCount, errorCount)
	return nil
}

// ValidatePNGSignature checks if a file has a valid PNG signature
func ValidatePNGSignature(data []byte) bool {
	return len(data) >= 8 && bytes.HasPrefix(data, pngSignature)
}

// fixPNGData attempts to fix PNG data by reconstructing missing signature
// This function works directly with byte arrays and is designed for use
// during the extraction process, similar to fixOggHeader
func fixPNGData(data []byte) ([]byte, error) {
	// Check if PNG is already valid
	if ValidatePNGSignature(data) {
		return data, nil
	}

	// Create a temporary PNGFixer to use existing logic
	fixer := &PNGFixer{
		OriginalData:   data,
		CorruptionInfo: make(map[string]any),
	}

	// Try to fix the PNG
	if err := fixer.FixPNG(); err != nil {
		return nil, err
	}

	// Return the fixed data
	return fixer.FixedData, nil
}

func fixOggHeader(data []byte) ([]byte, error) {
	const (
		sizeOfValidOggHeader = 16
		OggS                 = "OggS"
	)

	validHeader := []byte{byte(OggS[0]), byte(OggS[1]), byte(OggS[2]), byte(OggS[3]), 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // 14 bytes of valid Ogg header, last two bytes are unique identifier bytes
	headerFirstPart := data[:sizeOfValidOggHeader]
	if len(data) < sizeOfValidOggHeader {
		return nil, errors.New("not enough data, cannot fix Ogg header")
	}
	indexToCut, numberOfZeros := 0, 0
	for i := range headerFirstPart {
		for _, char := range OggS {
			if data[i] == byte(char) {
				continue // we got part of a OggS header or the 4 byte unique identifier, if the identifier has the oggs bytes we are fucked, this probably never happens
			}
		}
		if data[i] == 0x00 {
			numberOfZeros++
			if numberOfZeros > 9 {
				return nil, errors.New("too many zeros in the Ogg header, cannot fix")
			}
			continue
		}

		// we already checked if the current by is zero, so if we are here, we have a non-zero byte
		if i > 0 {
			if data[i-1] != 0x00 { // if the previous byte is not zero, this probably means we are in the unique identifier part of the header, its second byte
				if i > 2 && data[i-2] == 0x00 { // if we are in the second byte and the previous byte before the first one was zero, we are in the unique identifier part of the header
					indexToCut = i - 1
					break // we found the index to cut, we can break the loop
				}
			}
		}
	}
	dataWithoutHeader := data[indexToCut:]
	validHeader = append(validHeader, dataWithoutHeader...)
	return validHeader, nil
}
