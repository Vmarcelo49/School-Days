// GPK Batch Unpacker
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  Single file: go run . <gpk_file> [output_dir]")
		fmt.Println("  Batch mode:  go run . <directory_with_gpk_files> [output_dir]")
		fmt.Println("  Concurrent batch: go run . -c <directory_with_gpk_files> [output_dir]")
		fmt.Println("  Debug mode: go run . -debug <gpk_file>")
		fmt.Println("\nExamples:")
		fmt.Println("  go run . BGM.GPK")
		fmt.Println("  go run . \"D:\\Games\\Overflow\\SCHOOLDAYS HQ\\Packs\"")
		fmt.Println("  go run . -c \"D:\\Games\\Overflow\\SCHOOLDAYS HQ\\Packs\"")
		fmt.Println("  go run . -debug BGM.GPK")
		os.Exit(1)
	}

	var inputPath, outputDir string
	var useConcurrent, debugMode bool

	// Parse arguments
	if os.Args[1] == "-c" {
		useConcurrent = true
		if len(os.Args) < 3 {
			fmt.Println("Error: -c flag requires a directory path")
			os.Exit(1)
		}
		inputPath = os.Args[2]
		outputDir = "extracted"
		if len(os.Args) > 3 {
			outputDir = os.Args[3]
		}
	} else if os.Args[1] == "-debug" {
		debugMode = true
		if len(os.Args) < 3 {
			fmt.Println("Error: -debug flag requires a GPK file path")
			os.Exit(1)
		}
		inputPath = os.Args[2]
	} else {
		inputPath = os.Args[1]
		outputDir = "extracted"
		if len(os.Args) > 2 {
			outputDir = os.Args[2]
		}
	}

	// Debug mode - show compression information
	if debugMode {
		debugCompressionInfo(inputPath)
		return
	}

	// Check if input is a file or directory
	stat, err := os.Stat(inputPath)
	if err != nil {
		fmt.Printf("Error accessing path %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	if stat.IsDir() {
		// Batch mode - process all GPK files in directory
		if useConcurrent {
			fmt.Printf("Concurrent batch mode: Processing all GPK files in: %s\n", inputPath)
			err = processBatchConcurrent(inputPath, outputDir)
		} else {
			fmt.Printf("Batch mode: Processing all GPK files in: %s\n", inputPath)
			err = processBatch(inputPath, outputDir)
		}
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

// processBatchConcurrent processes multiple GPK files in parallel
func processBatchConcurrent(inputDir, outputDir string) error {
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

	// Determine optimal number of workers (don't exceed CPU cores for GPK-level parallelism)
	maxWorkers := runtime.NumCPU()
	if len(gpkFiles) < maxWorkers {
		maxWorkers = len(gpkFiles)
	}

	fmt.Printf("Processing %d GPK files using %d workers...\n\n", len(gpkFiles), maxWorkers)

	// Create channels for work distribution
	jobs := make(chan string, len(gpkFiles))
	results := make(chan error, len(gpkFiles))

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for gpkFile := range jobs {
				fmt.Printf("[Worker %d] Processing: %s\n", workerID, filepath.Base(gpkFile))

				// Create a subdirectory for this GPK file
				baseName := strings.TrimSuffix(filepath.Base(gpkFile), filepath.Ext(gpkFile))
				gpkOutputDir := filepath.Join(outputDir, baseName)

				err := processSingleFileConcurrent(gpkFile, gpkOutputDir)
				if err != nil {
					fmt.Printf("[Worker %d] Warning: Failed to process %s: %v\n", workerID, filepath.Base(gpkFile), err)
					results <- fmt.Errorf("failed to process %s: %w", filepath.Base(gpkFile), err)
				} else {
					fmt.Printf("[Worker %d] Completed: %s\n", workerID, filepath.Base(gpkFile))
					results <- nil
				}
			}
		}(w)
	}

	// Send jobs to workers
	for _, gpkFile := range gpkFiles {
		jobs <- gpkFile
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results and check for errors
	var errors []error
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		fmt.Printf("\nCompleted with %d errors out of %d files:\n", len(errors), len(gpkFiles))
		for _, err := range errors {
			fmt.Printf("  - %v\n", err)
		}
		return nil // Don't fail completely, just report errors
	}

	return nil
}

func processSingleFile(gpkFilePath, outputDir string) error {
	// Create GPK instance and load file
	gpk := NewGPK()
	err := gpk.Load(gpkFilePath)
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

// processSingleFileConcurrent processes a single GPK file with concurrent file extraction
func processSingleFileConcurrent(gpkFilePath, outputDir string) error {
	// Create GPK instance and load file
	gpk := NewGPK()
	err := gpk.Load(gpkFilePath)
	if err != nil {
		return fmt.Errorf("failed to load GPK file: %w", err)
	}

	entries := gpk.GetEntries()
	fmt.Printf("  Successfully loaded GPK file with %d entries\n", len(entries))

	// Extract all files using concurrent UnpackAll method
	fmt.Printf("  Extracting all %d files to: %s\n", len(entries), outputDir)

	err = gpk.UnpackAllConcurrent(outputDir)
	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	fmt.Printf("  Successfully extracted %d files\n", len(entries))
	return nil
}

// debugCompressionInfo shows compression information for GPK entries
func debugCompressionInfo(gpkFile string) {
	gpk := NewGPK()
	err := gpk.Load(gpkFile)
	if err != nil {
		fmt.Printf("Failed to load GPK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("GPK file: %s\n", gpkFile)
	fmt.Printf("Total entries: %d\n\n", len(gpk.GetEntries()))

	// Show compression info for first 10 entries
	fmt.Println("Compression information for first 10 entries:")
	fmt.Println("Name\t\t\t\tComprLen\tUncomprLen\tCompressed?")
	fmt.Println("================================================================================")

	for i, entry := range gpk.GetEntries() {
		if i >= 10 {
			break
		}

		compressed := "No"
		if entry.Header.UncomprLen > 0 && entry.Header.UncomprLen != entry.Header.ComprLen {
			compressed = "Yes"
		}

		fmt.Printf("%-30s\t%d\t\t%d\t\t%s\n",
			entry.Name,
			entry.Header.ComprLen,
			entry.Header.UncomprLen,
			compressed)
	}
}
