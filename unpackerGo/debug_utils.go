// Debug and investigation utilities for GPK and OGG file analysis
// This module consolidates debugging functions for analyzing file structure,
// investigating encoding issues, and testing fix processes.

package main

import (
	"fmt"
	"os"
)

// debugOGGCorruption performs comprehensive OGG corruption analysis
func debugOGGCorruption() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run . debug-ogg <ogg-file>")
		return
	}

	filename := os.Args[2]
	fmt.Printf("=== OGG DEBUG ANALYSIS for %s ===\n", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", len(data))

	// Show original hex dump
	fmt.Printf("\n--- ORIGINAL FILE HEX DUMP (first 128 bytes) ---\n")
	fmt.Print(HexDump(data, 128))

	// Analyze corruption patterns
	AnalyzeOGGCorruption(data)

	// Analyze structure
	analysis := AnalyzeOGGStructure(data)
	fmt.Printf("\n--- STRUCTURE ANALYSIS ---\n")
	fmt.Printf("Status: %v\n", analysis.Status)
	fmt.Printf("Description: %s\n", analysis.Description)
	fmt.Printf("First OggS position: %d\n", analysis.FirstOggSPos)
	fmt.Printf("Second OggS position: %d\n", analysis.SecondOggSPos)
	fmt.Printf("Vorbis position: %d\n", analysis.VorbisPos)
	fmt.Printf("Has valid structure: %v\n", analysis.HasValidStructure)

	// Test our fixing process
	fmt.Printf("\n--- TESTING FIX PROCESS ---\n")
	gpk := &GPK{}
	corrected := gpk.processOGGDataWithDebug(data, filename)

	// Validate the result
	fmt.Printf("\n--- VALIDATION ---\n")
	ValidateOGGFile(corrected, filename)

	// Save corrected file for testing
	correctedFilename := filename + ".fixed"
	err = os.WriteFile(correctedFilename, corrected, 0644)
	if err != nil {
		fmt.Printf("Error saving corrected file: %v\n", err)
	} else {
		fmt.Printf("Corrected file saved as: %s\n", correctedFilename)
	}
}

// investigateOGGEncoding analyzes OGG file encoding within GPK archives
func investigateOGGEncoding() {
	fmt.Println("=== Investigating OGG Data Encoding ===")

	// Open GPK file
	gpkFile, err := os.Open("BGM.GPK")
	if err != nil {
		fmt.Printf("Error opening GPK: %v\n", err)
		return
	}
	defer gpkFile.Close()

	// Load GPK
	gpk := NewGPK()
	err = gpk.Load("BGM.GPK")
	if err != nil {
		fmt.Printf("Error loading GPK: %v\n", err)
		return
	}

	// Test specific OGG file
	testFile := "SD_BGM/SDBGM01_INT.OGG"

	for _, entry := range gpk.GetEntries() {
		if entry.Name == testFile {
			fmt.Printf("File: %s\n", testFile)
			fmt.Printf("Offset: %d, ComprLen: %d, UncomprLen: %d, ComprHeadLen: %d\n",
				entry.Header.Offset, entry.Header.ComprLen,
				entry.Header.UncomprLen, entry.Header.ComprHeadLen)

			// Read larger chunk of raw data
			gpkFile.Seek(int64(entry.Header.Offset), 0)
			rawData := make([]byte, 200)
			n, _ := gpkFile.Read(rawData)

			fmt.Printf("\nFirst %d bytes of raw data:\n", n)
			for i := 0; i < n; i += 16 {
				end := i + 16
				if end > n {
					end = n
				}

				// Print hex
				fmt.Printf("%08X: ", i)
				for j := i; j < end; j++ {
					fmt.Printf("%02X ", rawData[j])
				}

				// Pad if needed
				for j := end; j < i+16; j++ {
					fmt.Printf("   ")
				}

				// Print ASCII
				fmt.Printf("|")
				for j := i; j < end; j++ {
					if rawData[j] >= 32 && rawData[j] <= 126 {
						fmt.Printf("%c", rawData[j])
					} else {
						fmt.Printf(".")
					}
				}
				fmt.Printf("|\n")
			}

			// Check for patterns
			fmt.Println("\n=== Pattern Analysis ===")
			analyzeCompressionPatterns(rawData)
			findOggSPatterns(rawData)
			analyzeHeaderStructure(rawData, entry)
			break
		}
	}
}

// analyzeCompressionPatterns looks for compression signatures in data
func analyzeCompressionPatterns(data []byte) {
	compressionSigs := []struct {
		sig  []byte
		name string
	}{
		{[]byte{0x78, 0x9C}, "zlib"},
		{[]byte{0x1F, 0x8B}, "gzip"},
		{[]byte{0x42, 0x5A}, "bzip2"},
		{[]byte{0x4C, 0x5A}, "lzma"},
	}

	for _, sigInfo := range compressionSigs {
		for i := 0; i <= len(data)-len(sigInfo.sig); i++ {
			match := true
			for j, b := range sigInfo.sig {
				if data[i+j] != b {
					match = false
					break
				}
			}
			if match {
				fmt.Printf("Found %s signature at offset %d: %X\n", sigInfo.name, i, sigInfo.sig)
			}
		}
	}
}

// findOggSPatterns locates OggS signatures in data
func findOggSPatterns(data []byte) {
	for i := 0; i <= len(data)-4; i++ {
		if string(data[i:i+4]) == "OggS" {
			fmt.Printf("Found 'OggS' at offset %d\n", i)
			if i+8 < len(data) {
				fmt.Printf("  Following bytes: %02X %02X %02X %02X\n",
					data[i+4], data[i+5], data[i+6], data[i+7])
			}
		}
	}
}

// analyzeHeaderStructure examines the header structure of GPK entries
func analyzeHeaderStructure(data []byte, entry GPKEntry) {
	fmt.Printf("\n--- Header Structure Analysis ---\n")
	fmt.Printf("Entry info: ComprHeadLen=%d\n", entry.Header.ComprHeadLen)

	if int(entry.Header.ComprHeadLen) < len(data) {
		headerData := data[:entry.Header.ComprHeadLen]
		fmt.Printf("Compression header (%d bytes):\n", len(headerData))
		printHexData(headerData)

		if int(entry.Header.ComprHeadLen) < len(data) {
			remainingData := data[entry.Header.ComprHeadLen:]
			fmt.Printf("\nData after compression header (first 32 bytes):\n")
			printHexData(remainingData[:min(32, len(remainingData))])
		}
	}
}

// printHexData prints data in hex format with ASCII representation
func printHexData(data []byte) {
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}

		// Print offset
		fmt.Printf("%04X: ", i)

		// Print hex
		for j := i; j < end; j++ {
			fmt.Printf("%02X ", data[j])
		}

		// Pad if needed
		for j := end; j < i+16; j++ {
			fmt.Printf("   ")
		}

		// Print ASCII
		fmt.Printf("|")
		for j := i; j < end; j++ {
			if data[j] >= 32 && data[j] <= 126 {
				fmt.Printf("%c", data[j])
			} else {
				fmt.Printf(".")
			}
		}
		fmt.Printf("|\n")
	}
}

// checkCompressionHeaders performs detailed compression header analysis
func checkCompressionHeaders() {
	// Parse the GPK file
	gpkFile, err := os.Open("BGM.GPK")
	if err != nil {
		fmt.Printf("Error opening GPK file: %v\n", err)
		return
	}
	defer gpkFile.Close()

	// Parse GPK
	gpk := NewGPK()
	err = gpk.Load("BGM.GPK")
	if err != nil {
		fmt.Printf("Error parsing GPK: %v\n", err)
		return
	}

	fmt.Println("=== Compression Header Analysis ===")

	// Check a few OGG files
	oggFiles := []string{
		"SD_BGM/SDBGM01_INT.OGG",
		"SD_BGM/SDBGM03_INT.OGG",
		"SD_BGM/SDBGM01_LOOP.OGG",
	}

	for _, filename := range oggFiles {
		for _, entry := range gpk.GetEntries() {
			if entry.Name == filename {
				fmt.Printf("\nFile: %s\n", filename)
				fmt.Printf("Offset: %d, ComprLen: %d, ComprHeadLen: %d\n",
					entry.Header.Offset, entry.Header.ComprLen, entry.Header.ComprHeadLen)

				// Read the compression header
				if entry.Header.ComprHeadLen > 0 {
					gpkFile.Seek(int64(entry.Header.Offset), 0)
					comprHeader := make([]byte, entry.Header.ComprHeadLen)
					gpkFile.Read(comprHeader)

					fmt.Printf("Compression header (%d bytes): ", entry.Header.ComprHeadLen)
					for i, b := range comprHeader {
						fmt.Printf("%02X", b)
						if i < len(comprHeader)-1 {
							fmt.Printf(" ")
						}
					}
					fmt.Printf("\n")

					// Check if it contains "OggS"
					if len(comprHeader) >= 4 {
						if string(comprHeader[0:4]) == "OggS" {
							fmt.Printf("*** COMPRESSION HEADER CONTAINS 'OggS'! ***\n")
						}
					}
				}

				// Also check the first 50 bytes at the original offset (including compression header)
				gpkFile.Seek(int64(entry.Header.Offset), 0)
				originalData := make([]byte, 50)
				gpkFile.Read(originalData)

				fmt.Printf("First 50 bytes at original offset: ")
				for i, b := range originalData {
					fmt.Printf("%02X", b)
					if i < len(originalData)-1 {
						fmt.Printf(" ")
					}
				}
				fmt.Printf("\n")

				// Look for "OggS" in the first 50 bytes
				for i := 0; i <= len(originalData)-4; i++ {
					if string(originalData[i:i+4]) == "OggS" {
						fmt.Printf("*** FOUND 'OggS' at position %d! ***\n", i)
					}
				}

				// Read the actual data after skipping compression header
				actualOffset := int64(entry.Header.Offset) + int64(entry.Header.ComprHeadLen)
				gpkFile.Seek(actualOffset, 0)
				actualData := make([]byte, 16)
				gpkFile.Read(actualData)

				fmt.Printf("Data after compression header (16 bytes): ")
				for i, b := range actualData {
					fmt.Printf("%02X", b)
					if i < len(actualData)-1 {
						fmt.Printf(" ")
					}
				}
				fmt.Printf("\n")

				// Check if actual data contains "OggS"
				if string(actualData[0:4]) == "OggS" {
					fmt.Printf("Data after header starts with 'OggS'\n")
				} else {
					fmt.Printf("Data after header does NOT start with 'OggS'\n")
				}

				break
			}
		}
	}
}
