package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// Vorbis identification header structure
type VorbisIdHeader struct {
	PacketType  uint8   // Should be 1 for identification header
	Identifier  [6]byte // "vorbis"
	Version     uint32  // Should be 0
	Channels    uint8   // Number of audio channels
	SampleRate  uint32  // Sample rate in Hz
	BitrateMax  uint32  // Maximum bitrate
	BitrateNom  uint32  // Nominal bitrate
	BitrateMin  uint32  // Minimum bitrate
	BlockSize   uint8   // Block size info
	FramingFlag uint8   // Framing flag (should be 1)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run validate_ogg.go <path_to_extracted_files>")
		fmt.Println("Example: go run validate_ogg.go extracted")
		os.Exit(1)
	}

	extractedPath := os.Args[1]

	fmt.Printf("Validating OGG Vorbis files in: %s\n", extractedPath)
	fmt.Println("=" + strings.Repeat("=", 60))

	var totalFiles, validFiles, invalidFiles int

	err := filepath.Walk(extractedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToUpper(filepath.Ext(info.Name())) == ".OGG" {
			totalFiles++
			fmt.Printf("\nValidating: %s\n", strings.Replace(path, extractedPath+string(filepath.Separator), "", 1))

			if validateOggVorbisFile(path) {
				validFiles++
				fmt.Println("✓ VALID: Proper OGG Vorbis file")
			} else {
				invalidFiles++
				fmt.Println("✗ INVALID: Not a proper OGG Vorbis file")
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("SUMMARY:\n")
	fmt.Printf("  Total files processed: %d\n", totalFiles)
	fmt.Printf("  Valid OGG Vorbis files: %d\n", validFiles)
	fmt.Printf("  Invalid files: %d\n", invalidFiles)

	if invalidFiles == 0 {
		fmt.Println("🎉 ALL FILES ARE VALID OGG VORBIS FILES!")
	} else {
		fmt.Printf("⚠️  %d files failed validation\n", invalidFiles)
	}
}

func validateOggVorbisFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("  Error opening file: %v\n", err)
		return false
	}
	defer file.Close()

	// Read and validate OGG page header
	var oggHeader OggPageHeader
	err = binary.Read(file, binary.LittleEndian, &oggHeader)
	if err != nil {
		fmt.Printf("  Error reading OGG header: %v\n", err)
		return false
	}

	// Check OGG capture pattern
	if string(oggHeader.CapturePattern[:]) != "OggS" {
		fmt.Printf("  Invalid OGG capture pattern: %s (expected: OggS)\n", string(oggHeader.CapturePattern[:]))
		return false
	}
	fmt.Printf("  ✓ OGG capture pattern: %s\n", string(oggHeader.CapturePattern[:]))

	// Check OGG version
	if oggHeader.Version != 0 {
		fmt.Printf("  Invalid OGG version: %d (expected: 0)\n", oggHeader.Version)
		return false
	}
	fmt.Printf("  ✓ OGG version: %d\n", oggHeader.Version)

	// Skip segment table
	segments := make([]byte, oggHeader.PageSegments)
	_, err = file.Read(segments)
	if err != nil {
		fmt.Printf("  Error reading segment table: %v\n", err)
		return false
	}

	// Calculate payload size
	var payloadSize int
	for _, segmentSize := range segments {
		payloadSize += int(segmentSize)
	}
	fmt.Printf("  ✓ Page segments: %d, Payload size: %d bytes\n", oggHeader.PageSegments, payloadSize)

	// Read the Vorbis identification header
	var vorbisHeader VorbisIdHeader
	err = binary.Read(file, binary.LittleEndian, &vorbisHeader)
	if err != nil {
		fmt.Printf("  Error reading Vorbis header: %v\n", err)
		return false
	}

	// Check packet type (should be 1 for identification header)
	if vorbisHeader.PacketType != 1 {
		fmt.Printf("  Invalid Vorbis packet type: %d (expected: 1)\n", vorbisHeader.PacketType)
		return false
	}
	fmt.Printf("  ✓ Vorbis packet type: %d\n", vorbisHeader.PacketType)

	// Check Vorbis identifier
	if string(vorbisHeader.Identifier[:]) != "vorbis" {
		fmt.Printf("  Invalid Vorbis identifier: %s (expected: vorbis)\n", string(vorbisHeader.Identifier[:]))
		return false
	}
	fmt.Printf("  ✓ Vorbis identifier: %s\n", string(vorbisHeader.Identifier[:]))

	// Check Vorbis version
	if vorbisHeader.Version != 0 {
		fmt.Printf("  Invalid Vorbis version: %d (expected: 0)\n", vorbisHeader.Version)
		return false
	}
	fmt.Printf("  ✓ Vorbis version: %d\n", vorbisHeader.Version)

	// Validate audio parameters
	if vorbisHeader.Channels == 0 || vorbisHeader.Channels > 255 {
		fmt.Printf("  Invalid channel count: %d\n", vorbisHeader.Channels)
		return false
	}
	fmt.Printf("  ✓ Channels: %d\n", vorbisHeader.Channels)

	if vorbisHeader.SampleRate == 0 {
		fmt.Printf("  Invalid sample rate: %d\n", vorbisHeader.SampleRate)
		return false
	}
	fmt.Printf("  ✓ Sample rate: %d Hz\n", vorbisHeader.SampleRate)

	// Check framing flag (should be 1)
	if vorbisHeader.FramingFlag != 1 {
		fmt.Printf("  Invalid framing flag: %d (expected: 1)\n", vorbisHeader.FramingFlag)
		return false
	}
	fmt.Printf("  ✓ Framing flag: %d\n", vorbisHeader.FramingFlag)

	fmt.Printf("  ✓ Bitrates - Max: %d, Nominal: %d, Min: %d\n",
		vorbisHeader.BitrateMax, vorbisHeader.BitrateNom, vorbisHeader.BitrateMin)

	return true
}
