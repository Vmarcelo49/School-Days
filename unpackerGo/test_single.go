package main

import (
	"fmt"
	"os"
)

// testSingleGPK tests a single GPK file and prints the first 5 entries
func testSingleGPK(gpkFile string) error {
	// Check if file exists
	if _, err := os.Stat(gpkFile); os.IsNotExist(err) {
		return fmt.Errorf("GPK file does not exist: %s", gpkFile)
	}

	fmt.Printf("Testing GPK file: %s\n", gpkFile)

	// Create GPK instance and try to load
	gpk := NewGPK()
	err := gpk.Load(gpkFile)
	if err != nil {
		return fmt.Errorf("failed to load GPK: %v", err)
	}

	fmt.Printf("Successfully parsed GPK file with %d entries\n", len(gpk.entries))

	// Print first few entries
	for i, entry := range gpk.entries {
		if i >= 5 { // Limit to first 5 as requested
			break
		}
		fmt.Printf("Entry %d: %s (Offset: %d, ComprLen: %d, UncomprLen: %d)\n",
			i+1, entry.Name, entry.Header.Offset, entry.Header.ComprLen, entry.Header.UncomprLen)
	}

	return nil
}
