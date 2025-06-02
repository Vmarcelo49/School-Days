// GPK Unpacker - Test and Debug Functions
package main

import (
	"fmt"
	"os"
)

// testOGGAnalysis runs OGG file analysis tests on predefined files
func testOGGAnalysis() {
	testFiles := []string{
		"extracted/SD_BGM/SDBGM12_LOOP.OGG",
		"extracted/SD_BGM/SDBGM10_INT.OGG",
		"extracted/VOCAL/SDV01.OGG",
		"extracted/SD_BGM/SDBGM01_INT.OGG",
		"extracted/SD_BGM/SDBGM01_LOOP.OGG",
	}

	fmt.Println("=== OGG Header Analysis Test ===")

	for _, filename := range testFiles {
		fmt.Printf("\nAnalyzing: %s\n", filename)

		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("  Error reading file: %v\n", err)
			continue
		}

		// Analyze the current structure
		analysis := AnalyzeOGGStructure(data)
		fmt.Printf("  Status: %v\n", analysis.Status)
		fmt.Printf("  Description: %s\n", analysis.Description)
		fmt.Printf("  First OggS at: %d\n", analysis.FirstOggSPos)
		fmt.Printf("  Second OggS at: %d\n", analysis.SecondOggSPos)
		fmt.Printf("  Vorbis at: %d\n", analysis.VorbisPos)
		fmt.Printf("  Has valid structure: %v\n", analysis.HasValidStructure)

		// Show first few bytes
		if len(data) >= 26 {
			fmt.Printf("  Header bytes 0-15: ")
			for i := 0; i < 16; i++ {
				fmt.Printf("%02X ", data[i])
			}
			fmt.Printf("\n")

			fmt.Printf("  Header bytes 16-25: ")
			for i := 16; i < 26; i++ {
				fmt.Printf("%02X ", data[i])
			}
			fmt.Printf("\n")

			// Extract specific fields
			serialNumber := data[14:18]
			crcChecksum := data[22:26]
			fmt.Printf("  Serial Number (bytes 14-17): %02X %02X %02X %02X\n",
				serialNumber[0], serialNumber[1], serialNumber[2], serialNumber[3])
			fmt.Printf("  CRC Checksum (bytes 22-25): %02X %02X %02X %02X\n",
				crcChecksum[0], crcChecksum[1], crcChecksum[2], crcChecksum[3])
		}
	}
}
