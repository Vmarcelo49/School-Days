// OGG Header Manipulation Functions
package main

import (
	"bytes"
)

// ExtractSerialNumberAtPosition extracts serial number from OGG data at specific position
func ExtractSerialNumberAtPosition(data []byte, oggSPos int) []byte {
	if oggSPos < 0 || len(data) < oggSPos+18 {
		return []byte{0x2A, 0x00, 0x00, 0x00} // Default from our analysis
	}
	return data[oggSPos+14 : oggSPos+18]
}

// ExtractCRCChecksum extracts the CRC checksum from original OGG data
// CRC is at bytes 22-25 of the first OGG page
func ExtractCRCChecksum(data []byte) []byte {
	oggSPos := bytes.Index(data, []byte("OggS"))
	if oggSPos == -1 || len(data) < oggSPos+26 {
		// Default CRC if not found
		return []byte{0x00, 0x00, 0x00, 0x00}
	}
	return data[oggSPos+22 : oggSPos+26]
}

// ExtractCRCChecksumAtPosition extracts CRC from OGG data at specific position
func ExtractCRCChecksumAtPosition(data []byte, oggSPos int) []byte {
	if oggSPos < 0 || len(data) < oggSPos+26 {
		return []byte{0x00, 0x00, 0x00, 0x00} // Default if not found
	}
	return data[oggSPos+22 : oggSPos+26]
}

// ExtractSerialNumber extracts the serial number from OGG data
// Serial number is at bytes 14-17 of the first OGG page
func ExtractSerialNumber(data []byte) []byte {
	oggSPos := bytes.Index(data, []byte("OggS"))
	if oggSPos == -1 || len(data) < oggSPos+18 {
		return []byte{0x2A, 0x00, 0x00, 0x00} // Default from our analysis
	}
	return data[oggSPos+14 : oggSPos+18]
}

// IsOGGHeaderCorrect checks if OGG header has the first OggS signature at position 0
func IsOGGHeaderCorrect(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	// Check first OggS at position 0
	return header[0] == 'O' && header[1] == 'g' && header[2] == 'g' && header[3] == 'S'
}

// FixOGGHeader attempts to fix the OGG header by adding missing first OggS signature
func FixOGGHeader(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, nil // Not enough data to fix
	}

	// Check if header is already correct
	if IsOGGHeaderCorrect(data) {
		return data, nil // Header is already correct
	}

	// Search for the OggS header in the data
	for i := 0; i <= len(data)-4; i++ {
		if data[i] == 'O' && data[i+1] == 'g' && data[i+2] == 'g' && data[i+3] == 'S' {
			// Found OggS header, add the missing first OggS
			fixedData := make([]byte, 0, len(data)+4)
			fixedData = append(fixedData, []byte("OggS")...)
			fixedData = append(fixedData, data[:i]...)
			fixedData = append(fixedData, data[i:]...)
			return fixedData, nil
		}
	}

	return nil, nil // No valid OggS header found
}
