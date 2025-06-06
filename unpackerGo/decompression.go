package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// isValidZlibHeader checks if the given 2 bytes form a valid zlib header
func isValidZlibHeader(b1, b2 byte) bool {
	// zlib header format:
	// First byte: CMF (Compression Method and flags)
	//   - bits 0-3: CM (compression method) - should be 8 for deflate
	//   - bits 4-7: CINFO (compression info) - window size
	// Second byte: FLG (flags)
	//   - bits 0-4: FCHECK - check bits for CMF and FLG
	//   - bit 5: FDICT - preset dictionary flag
	//   - bits 6-7: FLEVEL - compression level

	// Check compression method (should be 8 for deflate)
	if (b1 & 0x0F) != 8 {
		return false
	}

	// Check that the header passes the checksum test
	header := uint16(b1)<<8 | uint16(b2)
	return (header % 31) == 0
}

// decompressData decompresses the given data using zlib
func decompressData(compressedData []byte, uncompressedSize uint32) ([]byte, error) {
	if len(compressedData) < 4 {
		return nil, fmt.Errorf("compressed data too short: need at least 4 bytes, have %d", len(compressedData))
	}
	/*
		originalSize := binary.LittleEndian.Uint32(compressedData)

		// Verify the size matches what we expect
		if originalSize != uncompressedSize {
			return nil, fmt.Errorf("size mismatch: header says %d, expected %d", originalSize, uncompressedSize)
		}
	*/
	// Skip the 4-byte size header and start decompression from offset 4
	fmt.Printf("first 16 bytes of compressed data:%02x \n", compressedData[:16])
	fmt.Printf("compressed data length: %d, uncompressed size: %d\n", len(compressedData), uncompressedSize)
	fmt.Printf("Compressed data starts with: %02X %02X\n", compressedData[0], compressedData[1])
	highByte := byte((uncompressedSize >> 8) & 0xFF)
	lowByte := byte(uncompressedSize & 0xFF)
	fmt.Printf("Expected uncompressed size: %02X %02X\n", highByte, lowByte)
	zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData[4:]))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w ", err)
	}
	defer zlibReader.Close()
	fmt.Printf("Decompressing data with expected size: %x", uncompressedSize)
	// Decompress the data
	decompressedData, err := io.ReadAll(zlibReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}
	/*
		// Verify the decompressed size
		if len(decompressedData) != int(uncompressedSize) {
			return nil, fmt.Errorf("decompressed size mismatch: got %d bytes, expected %d", len(decompressedData), uncompressedSize)
		}
	*/
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
	var decryptionApplied bool

	// If forced to skip decryption, try original data first
	if forceSkipDecryption {
		// Check if we have a valid zlib header at offset 0
		if len(data) >= 2 && isValidZlibHeader(data[0], data[1]) {
			dataToDecompress = data
			decryptionApplied = false
		} else if len(data) >= 6 && isValidZlibHeader(data[4], data[5]) {
			dataToDecompress = data[4:]
			decryptionApplied = false
		}
	}

	// If we haven't found a valid header yet, try with decryption
	if dataToDecompress == nil {
		decryptData(data)
		decryptionApplied = true

		// Check if we have a valid zlib header at offset 0
		if len(data) >= 2 && isValidZlibHeader(data[0], data[1]) {
			dataToDecompress = data
		} else if len(data) >= 6 && isValidZlibHeader(data[4], data[5]) {
			dataToDecompress = data[4:]
		}
	}

	// If still no valid header, show debug info
	if dataToDecompress == nil {
		fmt.Printf("No valid zlib header found (decryption applied: %v)\n", decryptionApplied)
		if len(data) >= 6 {
			fmt.Printf("Bytes at offset 0: %02X %02X\n", data[0], data[1])
			fmt.Printf("Bytes at offset 4: %02X %02X\n", data[4], data[5])
		}
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
