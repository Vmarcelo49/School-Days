// GPK Batch Unpacker
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  Single file: go run . <gpk_file> [output_dir]")
		fmt.Println("  Batch mode:  go run . <directory_with_gpk_files> [output_dir]")
		fmt.Println("\nExamples:")
		fmt.Println("  go run . BGM.GPK")
		fmt.Println("  go run . \"D:\\Games\\Overflow\\SCHOOLDAYS HQ\\Packs\"")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputDir := "extracted"
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	// Check if input is a file or directory
	stat, err := os.Stat(inputPath)
	if err != nil {
		fmt.Printf("Error accessing path %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	if stat.IsDir() {
		// Batch mode - process all GPK files in directory
		fmt.Printf("Batch mode: Processing all GPK files in: %s\n", inputPath)
		err = processBatch(inputPath, outputDir)
	} else {
		// Single file mode
		fmt.Printf("Single file mode: Processing: %s\n", inputPath)
		err = processSingleFile(inputPath, outputDir)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nExtraction completed successfully!")
}

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

	fmt.Printf("Found %d GPK files to process:\n", len(gpkFiles))
	for i, file := range gpkFiles {
		fmt.Printf("  %d. %s\n", i+1, filepath.Base(file))
	}
	fmt.Println()

	// Process each GPK file
	for i, gpkFile := range gpkFiles {
		fmt.Printf("=== Processing %d/%d: %s ===\n", i+1, len(gpkFiles), filepath.Base(gpkFile))

		// Create a subdirectory for this GPK file
		baseName := strings.TrimSuffix(filepath.Base(gpkFile), filepath.Ext(gpkFile))
		gpkOutputDir := filepath.Join(outputDir, baseName)

		err := processSingleFile(gpkFile, gpkOutputDir)
		if err != nil {
			fmt.Printf("Warning: Failed to process %s: %v\n", filepath.Base(gpkFile), err)
			continue
		}

		fmt.Printf("Completed: %s\n\n", filepath.Base(gpkFile))
	}
	return nil
}

func processSingleFile(gpkFile, outputDir string) error {
	// Create GPK instance and load file
	gpk := NewGPK()
	err := gpk.Load(gpkFile)
	if err != nil {
		return fmt.Errorf("failed to load GPK file: %w", err)
	}

	entries := gpk.GetEntries()
	fmt.Printf("Successfully loaded GPK file with %d entries\n", len(entries))

	// Show first few entries
	fmt.Println("\nFirst 10 entries:")
	for i, entry := range entries {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d: %s (Offset: %d, Size: %d bytes)\n",
			i+1, entry.Name, entry.Header.Offset, entry.Header.ComprLen)
	}

	// Extract all files using UnpackAll method
	fmt.Printf("\nExtracting all %d files to: %s\n", len(entries), outputDir)

	err = gpk.UnpackAll(outputDir)
	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	fmt.Printf("Successfully extracted %d files\n", len(entries))
	return nil
}
