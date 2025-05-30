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
	"runtime"
	"strings"
	"sync"
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

// readGPKSignature reads GPK signature manually to ensure exact 32-byte layout
func readGPKSignature(reader io.Reader) (*GPKSignature, error) {
	sig := &GPKSignature{}

	// Read each field explicitly to ensure exact byte layout (32 bytes total)
	if err := binary.Read(reader, binary.LittleEndian, &sig.Sig0); err != nil {
		return nil, fmt.Errorf("failed to read Sig0: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &sig.PidxLength); err != nil {
		return nil, fmt.Errorf("failed to read PidxLength: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &sig.Sig1); err != nil {
		return nil, fmt.Errorf("failed to read Sig1: %w", err)
	}

	return sig, nil
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
	// Read signature from the end of file manually to avoid padding issues
	const signatureSize = 32 // 12 + 4 + 16 bytes exactly
	_, err = file.Seek(fileSize-signatureSize, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to signature: %w", err)
	}

	signature, err := readGPKSignature(file)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}

	// Verify signature
	if string(signature.Sig0[:len(GPKTailerIdent0)]) != GPKTailerIdent0 ||
		string(signature.Sig1[:len(GPKTailerIdent1)]) != GPKTailerIdent1 {
		return fmt.Errorf("invalid GPK signature")
	}
	// Read compressed PIDX data
	pidxOffset := fileSize - signatureSize - int64(signature.PidxLength)
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

// readGPKEntryHeader reads GPK entry header manually to ensure exact 23-byte layout
func readGPKEntryHeader(data []byte) (*GPKEntryHeader, error) {
	if len(data) < 23 {
		return nil, fmt.Errorf("insufficient data for header: need 23 bytes, have %d", len(data))
	}

	header := &GPKEntryHeader{}
	reader := bytes.NewReader(data)

	// Read each field in exact order to match C++ struct (total: 23 bytes)
	if err := binary.Read(reader, binary.LittleEndian, &header.SubVersion); err != nil { // 2 bytes
		return nil, fmt.Errorf("failed to read SubVersion: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.Version); err != nil { // 2 bytes
		return nil, fmt.Errorf("failed to read Version: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.Zero); err != nil { // 2 bytes
		return nil, fmt.Errorf("failed to read Zero: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.Offset); err != nil { // 4 bytes
		return nil, fmt.Errorf("failed to read Offset: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ComprLen); err != nil { // 4 bytes
		return nil, fmt.Errorf("failed to read ComprLen: %w", err)
	}
	if _, err := reader.Read(header.Dflt[:]); err != nil { // 4 bytes
		return nil, fmt.Errorf("failed to read Dflt: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.UncomprLen); err != nil { // 4 bytes
		return nil, fmt.Errorf("failed to read UncomprLen: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ComprHeadLen); err != nil { // 1 byte
		return nil, fmt.Errorf("failed to read ComprHeadLen: %w", err)
	}
	// Total: exactly 23 bytes

	return header, nil
}

// Parse parses the GPK file and extracts entries
func (g *GPK) Parse() error {
	file, err := os.Open(g.fileName)
	if err != nil {
		return fmt.Errorf("failed to open GPK file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}
	fileSize := stat.Size()

	// Read signature from the end of file manually to avoid padding issues
	const signatureSize = 32 // 12 + 4 + 16 bytes exactly
	_, err = file.Seek(fileSize-signatureSize, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to signature: %w", err)
	}

	signature, err := readGPKSignature(file)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}

	// Verify signature
	if string(signature.Sig0[:len(GPKTailerIdent0)]) != GPKTailerIdent0 ||
		string(signature.Sig1[:len(GPKTailerIdent1)]) != GPKTailerIdent1 {
		return fmt.Errorf("invalid GPK signature")
	}
	// Read compressed PIDX data
	pidxOffset := fileSize - signatureSize - int64(signature.PidxLength)
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

// decompressData decompresses Qt's qUncompress format data
// Based on the C++ decompress_data function in unpacker_cli_unfinished/gpk.cpp lines 64-86
func decompressData(compressedData []byte, uncompressedSize uint32) ([]byte, error) {
	if len(compressedData) < 4 {
		return nil, fmt.Errorf("compressed data too short: need at least 4 bytes, have %d", len(compressedData))
	}

	// Qt's qUncompress format: first 4 bytes contain uncompressed size in big-endian
	// followed by standard zlib data
	originalSize := binary.BigEndian.Uint32(compressedData[:4])

	// Verify the size matches what we expect
	if originalSize != uncompressedSize {
		return nil, fmt.Errorf("size mismatch: header says %d, expected %d", originalSize, uncompressedSize)
	}

	// Create zlib reader for the data after the 4-byte header
	zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData[4:]))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zlibReader.Close()

	// Decompress the data
	decompressedData, err := io.ReadAll(zlibReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	// Verify the decompressed size
	if len(decompressedData) != int(uncompressedSize) {
		return nil, fmt.Errorf("decompressed size mismatch: got %d bytes, expected %d", len(decompressedData), uncompressedSize)
	}

	return decompressedData, nil
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

		// Read header using the manual reading function to ensure exact 23-byte layout
		headerBytes := data[offset : offset+headerSize]
		offset += headerSize

		// Parse the header using the new manual reading function
		header, err := readGPKEntryHeader(headerBytes)
		if err != nil {
			return fmt.Errorf("failed to parse header: %w", err)
		}

		// Create entry with parsed header
		var entry GPKEntry
		entry.Name = filename
		entry.Header = *header

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
		// Check if this looks like a reasonable character (null character check only)
		if utf16Data[i] == 0 {
			return false
		}
	}

	possibleName := string(utf16.Decode(utf16Data))
	// Check if the name looks like a reasonable filename
	return strings.Contains(possibleName, "/") && strings.Contains(possibleName, ".")
}

// GetEntries returns all entries in the GPK
func (g *GPK) GetEntries() []GPKEntry {
	return g.entries
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

// UnpackAll unpacks all files in the GPK to the specified directory
func (g *GPK) UnpackAll(outputDir string) error {
	file, err := os.Open(g.fileName)
	if err != nil {
		return fmt.Errorf("failed to open GPK file: %w", err)
	}
	defer file.Close()
	for i, entry := range g.entries {
		fmt.Printf("Extracting %d/%d: %s\n", i+1, len(g.entries), entry.Name)

		// Use original filename directly (GPK files already have correct extensions)
		outputPath := filepath.Join(outputDir, entry.Name)
		outputDirPath := filepath.Dir(outputPath)

		err := os.MkdirAll(outputDirPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDirPath, err)
		}

		// Read file data - simple raw data extraction like C++ version
		_, err = file.Seek(int64(entry.Header.Offset), 0)
		if err != nil {
			return fmt.Errorf("failed to seek to entry %s: %w", entry.Name, err)
		}
		// Read raw data directly (matching C++ behavior: package.read(entry.header.comprlen))
		fileData := make([]byte, entry.Header.ComprLen)
		_, err = file.Read(fileData)
		if err != nil {
			return fmt.Errorf("failed to read entry %s: %w", entry.Name, err)
		}

		// Decompress file data if needed (when UncomprLen > 0 and differs from ComprLen)
		var finalData []byte
		if entry.Header.UncomprLen > 0 && entry.Header.UncomprLen != entry.Header.ComprLen {
			// File is compressed, decompress it using Qt's qUncompress format
			decompressedData, err := decompressData(fileData, entry.Header.UncomprLen)
			if err != nil {
				return fmt.Errorf("failed to decompress entry %s: %w", entry.Name, err)
			}
			finalData = decompressedData
		} else {
			// File is not compressed, use raw data
			finalData = fileData
		}

		// Strip OGG header if needed
		finalData = stripOGGHeader(finalData, entry.Name)

		// Write file
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
		}

		_, err = outFile.Write(finalData)
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}
	}

	return nil
}

// stripOGGHeader removes custom headers from OGG files, finding the "OggS" pattern
// and returning clean OGG data that starts with the actual OGG header
func stripOGGHeader(data []byte, filename string) []byte {
	// Look for OGG files by extension
	if !strings.HasSuffix(strings.ToUpper(filename), ".OGG") {
		return data
	}

	// Find the "OggS" pattern
	for i := 0; i < len(data)-3; i++ {
		if data[i] == 'O' && data[i+1] == 'g' && data[i+2] == 'g' && data[i+3] == 'S' {
			if i > 0 {
				// Found header to strip - return data starting from OggS
				return data[i:]
			} else {
				// Already starts with OggS, no header to strip
				return data
			}
		}
	}

	// No OggS pattern found, return original data
	return data
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

	// Simple wildcard matching without regex
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		// Convert wildcard pattern to regex
		escaped := regexp.QuoteMeta(pattern)
		escaped = strings.ReplaceAll(escaped, "\\*", ".*")
		escaped = strings.ReplaceAll(escaped, "\\?", ".")

		// Compile regex with case-insensitive flag
		regex, err := regexp.Compile("(?i)^" + escaped + "$")
		if err != nil {
			// If regex compilation fails, fall back to simple string matching
			return strings.EqualFold(pattern, name)
		}
		return regex.MatchString(name)
	}

	// Exact match (case insensitive)
	return strings.EqualFold(pattern, name)
}

// FileExtractionJob represents a single file extraction task
type FileExtractionJob struct {
	Entry      GPKEntry
	Index      int
	TotalFiles int
	OutputDir  string
}

// FileExtractionResult represents the result of a file extraction
type FileExtractionResult struct {
	Index    int
	Error    error
	Filename string
}

// UnpackAllConcurrent unpacks all files in the GPK to the specified directory using goroutines
func (g *GPK) UnpackAllConcurrent(outputDir string) error {
	// Determine optimal number of workers
	// For I/O intensive tasks, we can use more workers than CPU cores
	maxWorkers := runtime.NumCPU() * 2
	if len(g.entries) < maxWorkers {
		maxWorkers = len(g.entries)
	}

	// Limit to prevent too many file handles
	if maxWorkers > 10 {
		maxWorkers = 10
	}

	fmt.Printf("    Using %d workers for extracting %d files\n", maxWorkers, len(g.entries))

	// Create channels for work distribution
	jobs := make(chan FileExtractionJob, len(g.entries))
	results := make(chan FileExtractionResult, len(g.entries))

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			g.extractionWorker(workerID, jobs, results)
		}(w)
	}

	// Send jobs to workers
	for i, entry := range g.entries {
		jobs <- FileExtractionJob{
			Entry:      entry,
			Index:      i,
			TotalFiles: len(g.entries),
			OutputDir:  outputDir,
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results and check for errors
	var errors []error
	successCount := 0
	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("file %s: %w", result.Filename, result.Error))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		fmt.Printf("    Extraction completed with %d successes and %d errors\n", successCount, len(errors))
		for _, err := range errors {
			fmt.Printf("    Error: %v\n", err)
		}
		// Return first error, but continue processing
		return errors[0]
	}

	return nil
}

// extractionWorker processes file extraction jobs
func (g *GPK) extractionWorker(workerID int, jobs <-chan FileExtractionJob, results chan<- FileExtractionResult) {
	// Open the GPK file for this worker
	file, err := os.Open(g.fileName)
	if err != nil {
		// Send error for all jobs this worker would have processed
		for job := range jobs {
			results <- FileExtractionResult{
				Index:    job.Index,
				Error:    fmt.Errorf("worker %d failed to open GPK file: %w", workerID, err),
				Filename: job.Entry.Name,
			}
		}
		return
	}
	defer file.Close()

	for job := range jobs {
		fmt.Printf("    [Worker %d] Extracting %d/%d: %s\n",
			workerID, job.Index+1, job.TotalFiles, job.Entry.Name)

		err := g.extractSingleFile(file, job.Entry, job.OutputDir)
		results <- FileExtractionResult{
			Index:    job.Index,
			Error:    err,
			Filename: job.Entry.Name,
		}
	}
}

// normalizeFilename normalizes file names based on package type (like C++ version)
func normalizeFilename(packageName, originalName string) string {
	pkgUpper := strings.ToUpper(packageName)

	// Apply the same normalization rules as the C++ version
	if strings.HasPrefix(pkgUpper, "SYSSE") || strings.HasPrefix(pkgUpper, "SE") || strings.HasPrefix(pkgUpper, "VOICE") {
		return originalName + ".ogg"
	} else if strings.HasPrefix(pkgUpper, "BGM") {
		return originalName + "_loop.ogg"
	} else if strings.HasPrefix(pkgUpper, "EVENT") {
		return originalName + ".PNG"
	}

	return originalName
}

// extractSingleFile extracts a single file from the GPK (thread-safe version)
func (g *GPK) extractSingleFile(file *os.File, entry GPKEntry, outputDir string) error {
	// Use original filename directly (GPK files already have correct extensions)
	outputPath := filepath.Join(outputDir, entry.Name)
	outputDirPath := filepath.Dir(outputPath)

	err := os.MkdirAll(outputDirPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDirPath, err)
	}

	// Read file data - simple raw data extraction like C++ version
	_, err = file.Seek(int64(entry.Header.Offset), 0)
	if err != nil {
		return fmt.Errorf("failed to seek to entry %s: %w", entry.Name, err)
	}
	// Read raw data directly (matching C++ behavior: package.read(entry.header.comprlen))
	fileData := make([]byte, entry.Header.ComprLen)
	_, err = file.Read(fileData)
	if err != nil {
		return fmt.Errorf("failed to read entry %s: %w", entry.Name, err)
	}

	// Decompress file data if needed (when UncomprLen > 0 and differs from ComprLen)
	var finalData []byte
	if entry.Header.UncomprLen > 0 && entry.Header.UncomprLen != entry.Header.ComprLen {
		// File is compressed, decompress it using Qt's qUncompress format
		decompressedData, err := decompressData(fileData, entry.Header.UncomprLen)
		if err != nil {
			return fmt.Errorf("failed to decompress entry %s: %w", entry.Name, err)
		}
		finalData = decompressedData
	} else {
		// File is not compressed, use raw data
		finalData = fileData
	}

	// Strip OGG header if needed
	finalData = stripOGGHeader(finalData, entry.Name)

	// Write file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	_, err = outFile.Write(finalData)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}
	return nil
}
