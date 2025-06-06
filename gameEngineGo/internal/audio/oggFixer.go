package audio

import (
	"bytes"
	"fmt"
)

// fixOggHeader fixes corrupted OGG headers from GPK files
func (m *Manager) fixOggHeader(data []byte) ([]byte, error) {
	const (
		sizeOfValidOggHeader = 16
		OggS                 = "OggS"
	)

	if len(data) < sizeOfValidOggHeader {
		return nil, fmt.Errorf("not enough data, cannot fix Ogg header")
	}

	validHeader := []byte{
		byte(OggS[0]), byte(OggS[1]), byte(OggS[2]), byte(OggS[3]), // "OggS"
		0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Standard OGG header
	}

	// If the data already has a valid header, return as is
	if bytes.Equal(data[:sizeOfValidOggHeader], validHeader) {
		return data, nil
	}

	headerFirstPart := data[:sizeOfValidOggHeader]
	indexToCut, numberOfZeros := 0, 0

	for i := range headerFirstPart {
		// Skip OggS characters
		isOggChar := false
		for _, char := range OggS {
			if data[i] == byte(char) {
				isOggChar = true
				break
			}
		}
		if isOggChar {
			continue
		}

		if data[i] == 0x00 {
			numberOfZeros++
			if numberOfZeros > 9 {
				return nil, fmt.Errorf("too many zeros in the Ogg header, cannot fix")
			}
			continue
		}

		// Found non-zero byte
		if i > 0 {
			if data[i-1] != 0x00 { // Previous byte is not zero
				if i > 2 && data[i-2] == 0x00 { // Two bytes back was zero
					indexToCut = i - 1
					break
				}
			}
		}
	}

	dataWithoutHeader := data[indexToCut:]
	validHeader = append(validHeader, dataWithoutHeader...)
	return validHeader, nil
}
