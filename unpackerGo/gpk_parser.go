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

// GPKEntryHeader represents an entry header in the GPK file
type GPKEntryHeader struct {
	SubVersion        uint16
	Version           uint16  // Always 1?
	Zero              uint16  // Always 0
	Offset            uint32  // File data offset in GPK
	CompressedFileLen uint32  // Compressed file size
	MagicDFLT         [4]byte // reserved? magic "DFLT" value, can also be "    " not enough info on it // the original C++ code didnt use it anywhere
	UncompressedLen   uint32  // raw pidx data length(if magic isn't DFLT, then this filed always zero)
	comprheadlen      byte    // Variable compression header length // unused in the original C++ code, but present in the header
}

// GPKSignature represents the GPK file signature
type GPKSignature struct {
	Sig0       [12]byte
	PidxLength uint32
	Sig1       [16]byte
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

// FileExtractionJob and FileExtractionResult types are now defined in gpk_extraction.go
// for better module organization

// NewGPK creates a new GPK instance
func NewGPK() *GPK {
	return &GPK{
		entries: make([]GPKEntry, 0)}
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
	if err := binary.Read(reader, binary.LittleEndian, &header.CompressedFileLen); err != nil { // 4 bytes
		return nil, fmt.Errorf("failed to read ComprLen: %w", err)
	}
	if _, err := io.ReadFull(reader, header.MagicDFLT[:]); err != nil { // 4 bytes - padding/reserved space
		return nil, fmt.Errorf("failed to read Reserved field: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.UncompressedLen); err != nil { // 4 bytes
		return nil, fmt.Errorf("failed to read UncomprLen: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.comprheadlen); err != nil { // 1 byte
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
	signature, isAlreadyDecrypted, err := g.readSignature(file, fileSize)
	if err != nil {
		return err
	}

	// Read and decompress PIDX data
	uncompressedData, err := g.readPIDXData(file, fileSize, signature, isAlreadyDecrypted)
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
// Returns the signature and a boolean indicating if the file was already decrypted
func (g *GPK) readSignature(file *os.File, fileSize int64) (*GPKSignature, bool, error) {
	const signatureSize = 32 // 12 + 4 + 16 bytes exactly
	_, err := file.Seek(fileSize-signatureSize, 0)
	if err != nil {
		return nil, false, fmt.Errorf("failed to seek to signature: %w", err)
	}

	// Read raw signature data
	encryptedSig := make([]byte, signatureSize)
	_, err = file.Read(encryptedSig)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read signature: %w", err)
	}

	// Try decrypted signature first
	decryptedSig := make([]byte, signatureSize)
	copy(decryptedSig, encryptedSig)
	decryptData(decryptedSig)

	signature, err := readGPKSignature(bytes.NewReader(decryptedSig))
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse decrypted signature: %w", err)
	}

	// Check if decrypted signature is valid
	isValidDecrypted := string(signature.Sig0[:len(GPKTailerIdent0)]) == GPKTailerIdent0 &&
		string(signature.Sig1[:len(GPKTailerIdent1)]) == GPKTailerIdent1

	if isValidDecrypted {
		// File was encrypted, we had to decrypt the signature
		return signature, false, nil
	}

	// Try original signature (might be already decrypted)
	signatureOriginal, err := readGPKSignature(bytes.NewReader(encryptedSig))
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse original signature: %w", err)
	}

	isValidOriginal := string(signatureOriginal.Sig0[:len(GPKTailerIdent0)]) == GPKTailerIdent0 &&
		string(signatureOriginal.Sig1[:len(GPKTailerIdent1)]) == GPKTailerIdent1

	if isValidOriginal {
		// File was already decrypted
		return signatureOriginal, true, nil
	}

	return nil, false, fmt.Errorf("invalid GPK signature - neither encrypted nor decrypted version is valid")
}

// readPIDXData reads and decompresses the PIDX data from the GPK file
func (g *GPK) readPIDXData(file *os.File, fileSize int64, signature *GPKSignature, isAlreadyDecrypted bool) ([]byte, error) {
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

	// Decompress PIDX data with auto-detection, but hint from signature detection
	uncompressedData, err := decompressPIDX(compressedData, isAlreadyDecrypted)
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
		if header.comprheadlen > 0 {
			newOffset += int(header.comprheadlen)
		}

		offset = newOffset

		// Create and add entry
		entry := GPKEntry{
			Name:   filename,
			Header: *header,
		}
		g.entries = append(g.entries, entry)
		// Check for continuation or end of data
		if offset >= len(data)-2 {
			break
		}

		nextPotentialLen := binary.LittleEndian.Uint16(data[offset : offset+2])
		if nextPotentialLen == 0 || nextPotentialLen > 1024 {
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
