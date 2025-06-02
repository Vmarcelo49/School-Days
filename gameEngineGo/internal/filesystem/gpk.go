package filesystem

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

// GPK constants from the original unpacker
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
	SubVersion   uint16
	Version      uint16
	Zero         uint16
	Offset       uint32
	ComprLen     uint32
	Reserved     [4]byte
	UncomprLen   uint32
	ComprHeadLen uint8
}

// GPKEntry represents a file entry in the GPK package
type GPKEntry struct {
	Name   string
	Header GPKEntryHeader
}

// GPKSignature represents the GPK file signature
type GPKSignature struct {
	Sig0       [12]byte
	PidxLength uint32
	Sig1       [16]byte
}

// GPKArchive represents a GPK archive file
type GPKArchive struct {
	filename string
	entries  []GPKEntry
	file     *os.File
}

// NewGPKArchive creates a new GPK archive instance
func NewGPKArchive(filename string) *GPKArchive {
	return &GPKArchive{
		filename: filename,
		entries:  make([]GPKEntry, 0),
	}
}

// Open opens and parses the GPK archive
func (g *GPKArchive) Open() error {
	var err error
	g.file, err = os.Open(g.filename)
	if err != nil {
		return fmt.Errorf("failed to open GPK file %s: %v", g.filename, err)
	}

	// Get file size
	fileInfo, err := g.file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	fileSize := fileInfo.Size()

	// Read GPK signature from end of file
	if _, err := g.file.Seek(fileSize-32, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to signature: %v", err)
	}

	sig, err := g.readGPKSignature()
	if err != nil {
		return fmt.Errorf("failed to read GPK signature: %v", err)
	}

	// Validate signature
	if string(sig.Sig0[:]) != GPKTailerIdent0 || string(sig.Sig1[:]) != GPKTailerIdent1 {
		return fmt.Errorf("invalid GPK signature")
	}

	// Read PIDX data
	pidxOffset := fileSize - 32 - int64(sig.PidxLength)
	if _, err := g.file.Seek(pidxOffset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to PIDX: %v", err)
	}

	pidxData := make([]byte, sig.PidxLength)
	if _, err := g.file.Read(pidxData); err != nil {
		return fmt.Errorf("failed to read PIDX data: %v", err)
	}

	// Decrypt PIDX data
	g.decryptData(pidxData)

	// Parse entries
	if err := g.parseEntries(pidxData); err != nil {
		return fmt.Errorf("failed to parse entries: %v", err)
	}

	return nil
}

// Close closes the GPK archive
func (g *GPKArchive) Close() error {
	if g.file != nil {
		return g.file.Close()
	}
	return nil
}

// readGPKSignature reads the GPK signature
func (g *GPKArchive) readGPKSignature() (*GPKSignature, error) {
	sig := &GPKSignature{}

	if err := binary.Read(g.file, binary.LittleEndian, &sig.Sig0); err != nil {
		return nil, err
	}
	if err := binary.Read(g.file, binary.LittleEndian, &sig.PidxLength); err != nil {
		return nil, err
	}
	if err := binary.Read(g.file, binary.LittleEndian, &sig.Sig1); err != nil {
		return nil, err
	}

	return sig, nil
}

// decryptData decrypts data using the cipher code
func (g *GPKArchive) decryptData(data []byte) {
	for i := range data {
		data[i] ^= cipherCode[i%16]
	}
}

// parseEntries parses the entries from PIDX data
func (g *GPKArchive) parseEntries(pidxData []byte) error {
	reader := bytes.NewReader(pidxData)

	for reader.Len() > 0 {
		// Read entry header (23 bytes)
		headerData := make([]byte, 23)
		if n, err := reader.Read(headerData); err != nil || n != 23 {
			break // End of entries
		}

		header, err := g.readGPKEntryHeader(headerData)
		if err != nil {
			return err
		}

		// Read filename (UTF-16LE, null-terminated)
		filename, err := g.readUTF16String(reader)
		if err != nil {
			return err
		}

		// Add entry
		entry := GPKEntry{
			Name:   filename,
			Header: *header,
		}
		g.entries = append(g.entries, entry)
	}

	return nil
}

// readGPKEntryHeader reads a GPK entry header
func (g *GPKArchive) readGPKEntryHeader(data []byte) (*GPKEntryHeader, error) {
	if len(data) < 23 {
		return nil, fmt.Errorf("insufficient data for header")
	}

	header := &GPKEntryHeader{}
	reader := bytes.NewReader(data)

	binary.Read(reader, binary.LittleEndian, &header.SubVersion)
	binary.Read(reader, binary.LittleEndian, &header.Version)
	binary.Read(reader, binary.LittleEndian, &header.Zero)
	binary.Read(reader, binary.LittleEndian, &header.Offset)
	binary.Read(reader, binary.LittleEndian, &header.ComprLen)
	binary.Read(reader, binary.LittleEndian, &header.Reserved)
	binary.Read(reader, binary.LittleEndian, &header.UncomprLen)
	binary.Read(reader, binary.LittleEndian, &header.ComprHeadLen)

	return header, nil
}

// readUTF16String reads a null-terminated UTF-16LE string
func (g *GPKArchive) readUTF16String(reader *bytes.Reader) (string, error) {
	var utf16Data []uint16

	for {
		var char uint16
		if err := binary.Read(reader, binary.LittleEndian, &char); err != nil {
			return "", err
		}
		if char == 0 {
			break
		}
		utf16Data = append(utf16Data, char)
	}

	if len(utf16Data) == 0 {
		return "", nil
	}

	runes := utf16.Decode(utf16Data)
	return string(runes), nil
}

// GetEntries returns all entries in the archive
func (g *GPKArchive) GetEntries() []GPKEntry {
	return g.entries
}

// GetEntry finds an entry by name
func (g *GPKArchive) GetEntry(name string) (*GPKEntry, bool) {
	for _, entry := range g.entries {
		if strings.EqualFold(entry.Name, name) {
			return &entry, true
		}
	}
	return nil, false
}

// ExtractFile extracts a file from the archive
func (g *GPKArchive) ExtractFile(name string) ([]byte, error) {
	entry, found := g.GetEntry(name)
	if !found {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	// Seek to file data
	if _, err := g.file.Seek(int64(entry.Header.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to file data: %v", err)
	}

	// Read compressed data
	compressedData := make([]byte, entry.Header.ComprLen)
	if _, err := g.file.Read(compressedData); err != nil {
		return nil, fmt.Errorf("failed to read compressed data: %v", err)
	}

	// For now, return the raw compressed data
	// TODO: Implement proper decompression based on compression header
	return compressedData, nil
}
