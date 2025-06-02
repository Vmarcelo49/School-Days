package main

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
	SubVersion   uint16  // Always 0
	Version      uint16  // Always 0
	Zero         uint16  // Always 0
	Offset       uint32  // File data offset in GPK
	ComprLen     uint32  // Compressed file size
	Reserved     [4]byte // Padding/reserved space - always 0x20202020 (four ASCII spaces)
	UncomprLen   uint32  // Uncompressed size (always 0 - size unknown)
	ComprHeadLen uint8   // Variable compression header length
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
