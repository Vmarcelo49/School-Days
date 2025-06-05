package main

import "errors"

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
