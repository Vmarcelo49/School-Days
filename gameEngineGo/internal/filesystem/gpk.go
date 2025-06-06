package filesystem

import (
	"bytes"
	"compress/zlib"
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
	MagicDFLT         [4]byte // reserved? magic "DFLT" value
	UncompressedLen   uint32  // raw pidx data length
	comprheadlen      byte    // Variable compression header length
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
	file     *os.File
}

// NewGPK creates a new GPK instance
func NewGPK(fileName string) (*GPK, error) {
	gpk := &GPK{
		entries:  make([]GPKEntry, 0),
		fileName: fileName,
	}

	err := gpk.load()
	if err != nil {
		return nil, err
	}

	return gpk, nil
}

// Close closes the GPK file handle
func (g *GPK) Close() error {
	if g.file != nil {
		return g.file.Close()
	}
	return nil
}

// GetEntries returns all entries in the GPK
func (g *GPK) GetEntries() []GPKEntry {
	return g.entries
}

// FindEntry finds an entry by name (case-insensitive)
func (g *GPK) FindEntry(name string) (*GPKEntry, bool) {
	for _, entry := range g.entries {
		if equalsCaseInsensitive(entry.Name, name) {
			return &entry, true
		}
	}
	return nil, false
}

// ExtractFile extracts a file from the GPK and returns its data
func (g *GPK) ExtractFile(entry *GPKEntry) ([]byte, error) {
	if g.file == nil {
		file, err := os.Open(g.fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open GPK file: %w", err)
		}
		g.file = file
	}

	// Seek to file data
	_, err := g.file.Seek(int64(entry.Header.Offset), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to file data: %w", err)
	}

	// Read compressed data
	compressedData := make([]byte, entry.Header.CompressedFileLen)
	_, err = g.file.Read(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to read compressed data: %w", err)
	}

	// Check if file needs decompression
	magicStr := string(entry.Header.MagicDFLT[:4])
	if magicStr == "DFLT" && entry.Header.UncompressedLen > 0 {
		// File is compressed, decompress it
		return decompressData(compressedData)
	}

	// File is not compressed, return as-is
	return compressedData, nil
}

// load loads and parses the GPK file
func (g *GPK) load() error {
	file, err := os.Open(g.fileName)
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

// readSignature reads and verifies the GPK signature from the end of the file
func (g *GPK) readSignature(file *os.File, fileSize int64) (*GPKSignature, bool, error) {
	const signatureSize = 32
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
		return signatureOriginal, true, nil
	}

	return nil, false, fmt.Errorf("invalid GPK signature")
}

// readGPKSignature reads GPK signature manually to ensure exact 32-byte layout
func readGPKSignature(reader io.Reader) (*GPKSignature, error) {
	sig := &GPKSignature{}

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

	// Decompress PIDX data
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
	headerSize := 23

	for offset < dataLen {
		// Check if we have at least 2 bytes for filename length
		if offset+2 > dataLen {
			break
		}

		// Read filename length
		filenameLen := binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2

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
	if offset+int(filenameLen)*2 > len(data) {
		return "", offset, fmt.Errorf("not enough data for filename: need %d bytes, have %d", filenameLen*2, len(data)-offset)
	}

	// Read filename (UTF-16LE)
	filenameBytes := data[offset : offset+int(filenameLen)*2]
	offset += int(filenameLen) * 2
	// Convert UTF-16LE to string
	utf16Data := make([]uint16, filenameLen)
	for i := 0; i < int(filenameLen); i++ {
		utf16Data[i] = binary.LittleEndian.Uint16(filenameBytes[i*2 : i*2+2])
	}
	filename := string(utf16.Decode(utf16Data))

	return filename, offset, nil
}

// parseHeader extracts the entry header from the data
func (g *GPK) parseHeader(data []byte, offset int, headerSize int) (*GPKEntryHeader, int, error) {
	if offset+headerSize > len(data) {
		return nil, offset, fmt.Errorf("not enough data for header: need %d bytes, have %d", headerSize, len(data)-offset)
	}

	headerBytes := data[offset : offset+headerSize]
	offset += headerSize

	header, err := readGPKEntryHeader(headerBytes)
	if err != nil {
		return nil, offset, fmt.Errorf("failed to parse header: %w", err)
	}

	return header, offset, nil
}

// readGPKEntryHeader reads GPK entry header manually to ensure exact 23-byte layout
func readGPKEntryHeader(data []byte) (*GPKEntryHeader, error) {
	if len(data) < 23 {
		return nil, fmt.Errorf("insufficient data for header: need 23 bytes, have %d", len(data))
	}

	header := &GPKEntryHeader{}
	reader := bytes.NewReader(data)

	if err := binary.Read(reader, binary.LittleEndian, &header.SubVersion); err != nil {
		return nil, fmt.Errorf("failed to read SubVersion: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.Version); err != nil {
		return nil, fmt.Errorf("failed to read Version: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.Zero); err != nil {
		return nil, fmt.Errorf("failed to read Zero: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.Offset); err != nil {
		return nil, fmt.Errorf("failed to read Offset: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.CompressedFileLen); err != nil {
		return nil, fmt.Errorf("failed to read ComprLen: %w", err)
	}
	if _, err := io.ReadFull(reader, header.MagicDFLT[:]); err != nil {
		return nil, fmt.Errorf("failed to read Reserved field: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.UncompressedLen); err != nil {
		return nil, fmt.Errorf("failed to read UncomprLen: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.comprheadlen); err != nil {
		return nil, fmt.Errorf("failed to read ComprHeadLen: %w", err)
	}

	return header, nil
}

// isValidZlibHeader checks if the given 2 bytes form a valid zlib header
func isValidZlibHeader(b1, b2 byte) bool {
	// Check compression method (should be 8 for deflate)
	if (b1 & 0x0F) != 8 {
		return false
	}

	// Check that the header passes the checksum test
	header := uint16(b1)<<8 | uint16(b2)
	return (header % 31) == 0
}

// decompressData decompresses the given data using zlib
func decompressData(compressedData []byte) ([]byte, error) {
	if len(compressedData) < 4 {
		return nil, fmt.Errorf("compressed data too short: need at least 4 bytes, have %d", len(compressedData))
	}

	// Skip the 4-byte size header and start decompression from offset 4
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

	return decompressedData, nil
}

// decryptData decrypts the given data using the cipher code
func decryptData(data []byte) {
	for i := range data {
		data[i] ^= cipherCode[i%16]
	}
}

// decompressPIDX decompresses PIDX data, with auto-detection of encryption status
func decompressPIDX(compressedData []byte, forceSkipDecryption bool) ([]byte, error) {
	// Make a copy to avoid modifying the original data
	data := make([]byte, len(compressedData))
	copy(data, compressedData)
	var dataToDecompress []byte

	// If forced to skip decryption, try original data first
	if forceSkipDecryption {
		// Check if we have a valid zlib header at offset 0
		if len(data) >= 2 && isValidZlibHeader(data[0], data[1]) {
			dataToDecompress = data
		} else if len(data) >= 6 && isValidZlibHeader(data[4], data[5]) {
			dataToDecompress = data[4:]
		}
	}

	// If we haven't found a valid header yet, try with decryption
	if dataToDecompress == nil {
		decryptData(data)

		// Check if we have a valid zlib header at offset 0
		if len(data) >= 2 && isValidZlibHeader(data[0], data[1]) {
			dataToDecompress = data
		} else if len(data) >= 6 && isValidZlibHeader(data[4], data[5]) {
			dataToDecompress = data[4:]
		}
	}

	// If still no valid header, return error
	if dataToDecompress == nil {
		return nil, fmt.Errorf("no valid zlib header found in PIDX data")
	}

	zlibReader, err := zlib.NewReader(bytes.NewReader(dataToDecompress))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zlibReader.Close()

	uncompressedData, err := io.ReadAll(zlibReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return uncompressedData, nil
}

// equalsCaseInsensitive compares two strings case-insensitively
func equalsCaseInsensitive(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
