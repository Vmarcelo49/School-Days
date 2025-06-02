// GPK Batch Processing Logic
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// processBatch handles processing multiple GPK files in a directory
func processBatch(inputDir, outputDir string) error {
	// Find all GPK files in the directory
	var gpkFiles []string

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToUpper(filepath.Ext(info.Name())) == ".GPK" {
			gpkFiles = append(gpkFiles, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(gpkFiles) == 0 {
		return fmt.Errorf("no GPK files found in directory: %s", inputDir)
	}
	InfoPrintf("Found %d GPK files to process:\n", len(gpkFiles))
	for i, file := range gpkFiles {
		VerbosePrintf(LogVerbose, "  %d. %s\n", i+1, filepath.Base(file))
	}
	VerbosePrintln(LogVerbose)

	// Process each GPK file
	for i, gpkFile := range gpkFiles {
		InfoPrintf("=== Processing %d/%d: %s ===\n", i+1, len(gpkFiles), filepath.Base(gpkFile))

		// Create a subdirectory for this GPK file
		baseName := strings.TrimSuffix(filepath.Base(gpkFile), filepath.Ext(gpkFile))
		gpkOutputDir := filepath.Join(outputDir, baseName)
		err := processSingleFile(gpkFile, gpkOutputDir)
		if err != nil {
			ErrorPrintf("Warning: Failed to process %s: %v\n", filepath.Base(gpkFile), err)
			continue
		}

		ResultPrintf("Completed: %s\n\n", filepath.Base(gpkFile))
	}
	return nil
}

// processSingleFile handles processing a single GPK file
func processSingleFile(gpkFilePath, outputDir string) error {
	// Create GPK instance and load file
	gpk := NewGPK()
	err := gpk.Load(gpkFilePath)
	if err != nil {
		return fmt.Errorf("failed to load GPK file: %w", err)
	}
	entries := gpk.GetEntries()
	InfoPrintf("Successfully loaded GPK file with %d entries\n", len(entries))

	// Show first few entries
	VerbosePrintln(LogVerbose, "\nFirst 10 entries:")
	for i, entry := range entries {
		if i >= 10 {
			break
		}
		VerbosePrintf(LogVerbose, "  %d: %s (Offset: %d, Size: %d bytes)\n",
			i+1, entry.Name, entry.Header.Offset, entry.Header.ComprLen)
	}

	// Extract all files using UnpackAll method (now concurrent by default)
	InfoPrintf("\nExtracting all %d files to: %s\n", len(entries), outputDir)
	err = gpk.UnpackAll(outputDir)
	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	ResultPrintf("Successfully extracted %d files\n", len(entries))

	return nil
}
