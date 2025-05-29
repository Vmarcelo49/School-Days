package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// OGG page header structure
type OggPageHeader struct {
	CapturePattern  [4]byte // "OggS"
	Version         uint8
	HeaderType      uint8
	GranulePosition uint64
	SerialNumber    uint32
	PageSequence    uint32
	CRC32Checksum   uint32
	PageSegments    uint8
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/analyze/main.go <ogg_file>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	fmt.Printf("Analyzing OGG stream: %s\n", filePath)
	fmt.Println("============================================================")

	pageNum := 0
	var offset int64 = 0

	for {
		// Read OGG page header
		var oggHeader OggPageHeader
		err = binary.Read(file, binary.LittleEndian, &oggHeader)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Printf("Error reading OGG header at offset %d: %v\n", offset, err)
			break
		}

		// Check if this is a valid OGG page
		if string(oggHeader.CapturePattern[:]) != "OggS" {
			fmt.Printf("Invalid OGG page at offset %d\n", offset)
			break
		}

		pageNum++
		fmt.Printf("\n--- Page %d (offset %d) ---\n", pageNum, offset)
		fmt.Printf("Capture Pattern: %s\n", string(oggHeader.CapturePattern[:]))
		fmt.Printf("Version: %d\n", oggHeader.Version)
		fmt.Printf("Header Type: 0x%02X", oggHeader.HeaderType)

		// Decode header type flags
		if oggHeader.HeaderType&0x01 != 0 {
			fmt.Printf(" (continued packet)")
		}
		if oggHeader.HeaderType&0x02 != 0 {
			fmt.Printf(" (first page)")
		}
		if oggHeader.HeaderType&0x04 != 0 {
			fmt.Printf(" (last page)")
		}
		fmt.Println()

		fmt.Printf("Granule Position: %d\n", oggHeader.GranulePosition)
		fmt.Printf("Serial Number: %d\n", oggHeader.SerialNumber)
		fmt.Printf("Page Sequence: %d\n", oggHeader.PageSequence)
		fmt.Printf("CRC32: 0x%08X\n", oggHeader.CRC32Checksum)
		fmt.Printf("Page Segments: %d\n", oggHeader.PageSegments)

		// Read segment table
		segments := make([]byte, oggHeader.PageSegments)
		_, err = file.Read(segments)
		if err != nil {
			fmt.Printf("Error reading segment table: %v\n", err)
			break
		}

		// Calculate payload size and show segment table
		var payloadSize int
		fmt.Printf("Segment Table: ")
		for i, segmentSize := range segments {
			fmt.Printf("%d", segmentSize)
			if i < len(segments)-1 {
				fmt.Printf(", ")
			}
			payloadSize += int(segmentSize)
		}
		fmt.Printf("\nPayload Size: %d bytes\n", payloadSize)

		// If this is the first page, try to identify Vorbis packet type
		if pageNum == 1 && payloadSize > 0 {
			packetStart := make([]byte, min(64, payloadSize))
			currentPos, _ := file.Seek(0, 1) // Get current position

			_, err = file.Read(packetStart)
			if err != nil {
				fmt.Printf("Error reading packet data: %v\n", err)
			} else {
				fmt.Printf("\nFirst packet analysis:\n")
				if len(packetStart) > 0 {
					fmt.Printf("Packet Type: %d (0x%02X)\n", packetStart[0], packetStart[0])

					switch packetStart[0] {
					case 1:
						fmt.Println("-> Vorbis Identification Header")
					case 3:
						fmt.Println("-> Vorbis Comment Header")
					case 5:
						fmt.Println("-> Vorbis Setup Header")
					default:
						if packetStart[0]%2 == 0 {
							fmt.Println("-> Audio Packet")
						} else {
							fmt.Println("-> Unknown Header Packet")
						}
					}
				}

				// Check for "vorbis" identifier
				if len(packetStart) >= 7 {
					if string(packetStart[1:7]) == "vorbis" {
						fmt.Printf("Vorbis Identifier: Found\n")
					} else {
						fmt.Printf("Vorbis Identifier: NOT FOUND (got: %q)\n", string(packetStart[1:7]))
					}
				}

				// Show hex dump of first 32 bytes
				fmt.Printf("Hex dump (first 32 bytes):\n")
				for i := 0; i < min(32, len(packetStart)); i += 16 {
					fmt.Printf("%04X: ", i)
					for j := i; j < min(i+16, len(packetStart)); j++ {
						fmt.Printf("%02X ", packetStart[j])
					}
					fmt.Printf("  ")
					for j := i; j < min(i+16, len(packetStart)); j++ {
						if packetStart[j] >= 32 && packetStart[j] < 127 {
							fmt.Printf("%c", packetStart[j])
						} else {
							fmt.Printf(".")
						}
					}
					fmt.Println()
				}
			}

			// Restore file position
			file.Seek(currentPos, 0)
		}

		// Skip payload
		_, err = file.Seek(int64(payloadSize), 1)
		if err != nil {
			fmt.Printf("Error skipping payload: %v\n", err)
			break
		}

		// Update offset for next page
		offset, _ = file.Seek(0, 1)

		// Only analyze first few pages to avoid too much output
		if pageNum >= 3 {
			fmt.Printf("\n... (stopping after first 3 pages)\n")
			break
		}
	}

	fmt.Printf("\nTotal pages analyzed: %d\n", pageNum)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
