// GPK file parsing functionality
// This module handles parsing GPK file structure, reading signatures,
// decompressing PIDX data, and extracting file entries.

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

// GPKSignature represents the GPK file signature
type GPKSignature struct {
	Sig0       [12]byte
	PidxLength uint32
	Sig1       [16]byte
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

// readGPKEntryHeader reads GPK entry header manually to ensure exact 23-byte layout
func readGPKEntryHeader(data []byte) (*GPKEntryHeader, error) {
	if len(data) < 23 {
		return nil, fmt.Errorf("insufficient data for header: need 23 bytes, have %d", len(data))
	}

	header := &GPKEntryHeader{}
	reader := bytes.NewReader(data)

	// Read each field in exact order to ensure correct size of 23 bytes
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
	if _, err := io.ReadFull(reader, header.Reserved[:]); err != nil { // 4 bytes - padding/reserved space
		return nil, fmt.Errorf("failed to read Reserved field: %w", err)
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

	// Read and verify signature
	signature, err := g.readSignature(file, fileSize)
	if err != nil {
		return err
	}

	// Read and decompress PIDX data
	uncompressedData, err := g.readPIDXData(file, fileSize, signature)
	if err != nil {
		return err
	}

	// Parse entries
	err = g.parseEntries(uncompressedData)
	if err != nil {
		return fmt.Errorf("failed to parse entries: %w", err)
	}

	return nil
}

// Parse parses the GPK file and extracts entries (alias for Load for backward compatibility)
func (g *GPK) Parse() error {
	return g.Load(g.fileName)
}

// readSignature reads and verifies the GPK signature from the end of the file
func (g *GPK) readSignature(file *os.File, fileSize int64) (*GPKSignature, error) {
	const signatureSize = 32 // 12 + 4 + 16 bytes exactly
	_, err := file.Seek(fileSize-signatureSize, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to signature: %w", err)
	}

	signature, err := readGPKSignature(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Verify signature
	if string(signature.Sig0[:len(GPKTailerIdent0)]) != GPKTailerIdent0 ||
		string(signature.Sig1[:len(GPKTailerIdent1)]) != GPKTailerIdent1 {
		return nil, fmt.Errorf("invalid GPK signature")
	}

	return signature, nil
}

// readPIDXData reads and decompresses the PIDX data from the GPK file
func (g *GPK) readPIDXData(file *os.File, fileSize int64, signature *GPKSignature) ([]byte, error) {
	const signatureSize = 32
	pidxOffset := fileSize - signatureSize - int64(signature.PidxLength)

	_, err := file.Seek(pidxOffset, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to PIDX data: %w", err)
	}

	compressedData := make([]byte, signature.PidxLength)
	_, err = file.Read(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to read compressed data: %w", err)
	}

	// Decompress PIDX data
	uncompressedData, err := decompressPIDX(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress PIDX data: %w", err)
	}

	return uncompressedData, nil
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

		// Parse filename
		filename, newOffset, err := g.parseFilename(data, offset, filenameLen)
		if err != nil {
			return err
		}
		offset = newOffset

		// Parse header
		header, newOffset, err := g.parseHeader(data, offset, headerSize)
		if err != nil {
			return err
		}

		// Skip the compression header if present
		if header.ComprHeadLen > 0 {
			newOffset += int(header.ComprHeadLen)
		}

		offset = newOffset

		// Create and add entry
		entry := GPKEntry{
			Name:   filename,
			Header: *header,
		}
		g.entries = append(g.entries, entry)

		// Check for continuation or end of data
		if !g.shouldContinueParsing(data, offset) {
			break
		}
	}

	return nil
}

// parseFilename extracts and converts UTF-16LE filename to string
func (g *GPK) parseFilename(data []byte, offset int, filenameLen uint16) (string, int, error) {
	// Check if we have enough data remaining for filename
	if offset+int(filenameLen)*2 > len(data) {
		return "", offset, fmt.Errorf("not enough data for filename: need %d bytes, have %d", filenameLen*2, len(data)-offset)
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

	return filename, offset, nil
}

// parseHeader extracts the entry header from the data
// we send the current offset of the filename and the header size of 23 bytes
func (g *GPK) parseHeader(data []byte, offset int, headerSize int) (*GPKEntryHeader, int, error) {
	// Check if we have enough data for header
	if offset+headerSize > len(data) {
		return nil, offset, fmt.Errorf("not enough data for header: need %d bytes, have %d", headerSize, len(data)-offset)
	}

	// Read header using the manual reading function to ensure exact 23-byte layout
	headerBytes := data[offset : offset+headerSize]
	offset += headerSize

	// Parse the header
	header, err := readGPKEntryHeader(headerBytes)
	if err != nil {
		return nil, offset, fmt.Errorf("failed to parse header: %w", err)
	}

	return header, offset, nil
}

// shouldContinueParsing determines if parsing should continue based on data patterns
func (g *GPK) shouldContinueParsing(data []byte, offset int) bool {
	if offset >= len(data)-2 {
		return false
	}

	nextPotentialLen := binary.LittleEndian.Uint16(data[offset : offset+2])
	if nextPotentialLen == 0 {
		return false
	}

	if nextPotentialLen > 1024 {
		// Look for entry separator patterns and try to recover
		return g.findNextValidEntry(data, offset)
	}

	return true
}

// findNextValidEntry attempts to find the next valid entry in case of parsing issues
func (g *GPK) findNextValidEntry(data []byte, offset int) bool {
	// Look for entry separator patterns: "OggS", "Ogg" + length, "Og" + length
	for tryOffset := offset; tryOffset < offset+10 && tryOffset < len(data)-4; tryOffset++ {
		// Check for "OggS" pattern
		if tryOffset+4 <= len(data) &&
			data[tryOffset] == 79 && data[tryOffset+1] == 103 &&
			data[tryOffset+2] == 103 && data[tryOffset+3] == 83 { // "OggS"

			// Look for next filename length after OggS + padding
			for nextOffset := tryOffset + 4; nextOffset < tryOffset+10 && nextOffset < len(data)-2; nextOffset++ {
				tryLen := binary.LittleEndian.Uint16(data[nextOffset : nextOffset+2])
				if tryLen > 0 && tryLen <= 100 && g.isValidFilename(data, nextOffset, tryLen) {
					return true
				}
			}
		}

		// Check for "Ogg" + length pattern
		if tryOffset+4 <= len(data) &&
			data[tryOffset] == 79 && data[tryOffset+1] == 103 &&
			data[tryOffset+2] == 103 { // "Ogg"

			potentialLen := data[tryOffset+3]
			if potentialLen > 0 && potentialLen <= 100 {
				nextOffset := tryOffset + 3
				if g.isValidFilename(data, nextOffset, uint16(potentialLen)) {
					return true
				}
			}
		}

		// Check for "Og" + length pattern
		if tryOffset+3 <= len(data) && data[tryOffset] == 79 && data[tryOffset+1] == 103 { // "Og"
			potentialLen := data[tryOffset+2]
			if potentialLen > 0 && potentialLen <= 100 {
				nextOffset := tryOffset + 2
				if g.isValidFilename(data, nextOffset, uint16(potentialLen)) {
					return true
				}
			}
		}
	}

	return false
}

// isValidFilename checks if the data at the given offset contains a valid UTF-16 filename
func (g *GPK) isValidFilename(data []byte, offset int, filenameLen uint16) bool {
	if offset+2+int(filenameLen)*2 >= len(data) {
		return false
	}

	nameBytes := data[offset+2 : offset+2+int(filenameLen)*2]
	utf16Data := make([]uint16, filenameLen)
	for i := range int(filenameLen) {
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
