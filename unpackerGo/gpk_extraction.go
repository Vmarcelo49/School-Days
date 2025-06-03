// GPK file extraction functionality
// This module handles the concurrent extraction of files from GPK archives.
//
// CRITICAL UPDATE (Based on analysis):
// OGG files in GPK archives have custom compression headers that must be stripped
// to produce playable audio files. The ComprHeadLen field in GPK entry headers
// specifies how many bytes to skip at the beginning of each file.
//
// Key findings:
// 1. ComprHeadLen contains the number of compression header bytes (typically 3-5)
// 2. After skipping these bytes, we need to find the actual "OggS" signature
// 3. The original C++ engine handles this transparently through the Stream class
// 4. Our extractor must manually skip these headers to produce clean OGG files
//
// This implementation correctly handles ComprHeadLen to extract playable OGG files.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// FileExtractionJob represents a job for extracting a single file
type FileExtractionJob struct {
	Entry      GPKEntry
	Index      int
	TotalFiles int
	OutputDir  string
}

// FileExtractionResult represents the result of a file extraction job
type FileExtractionResult struct {
	Index    int
	Error    error
	Filename string
}

// UnpackAll unpacks all files in the GPK to the specified directory using goroutines
func (g *GPK) UnpackAll(outputDir string) error {
	maxWorkers := min(min(len(g.entries), runtime.NumCPU()*2), 10)

	VerbosePrintf(LogVerbose, "    Using %d workers for extracting %d files\n", maxWorkers, len(g.entries))

	jobs := make(chan FileExtractionJob, len(g.entries))
	results := make(chan FileExtractionResult, len(g.entries))

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := range maxWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			g.extractionWorker(workerID, jobs, results)
		}(w)
	}

	// Send jobs to workers
	for i, entry := range g.entries {
		jobs <- FileExtractionJob{
			Entry:      entry,
			Index:      i,
			TotalFiles: len(g.entries),
			OutputDir:  outputDir,
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results and check for errors
	var errors []error
	successCount := 0
	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("file %s: %w", result.Filename, result.Error))
		} else {
			successCount++
		}
	}
	if len(errors) > 0 {
		ResultPrintf("    Extraction completed with %d successes and %d errors\n", successCount, len(errors))
		for _, err := range errors {
			ErrorPrintf("    Error: %v\n", err)
		}
		// Return first error, but continue processing
		return errors[0]
	}

	return nil
}

// extractionWorker processes file extraction jobs
func (g *GPK) extractionWorker(workerID int, jobs <-chan FileExtractionJob, results chan<- FileExtractionResult) {
	// Open the GPK file for this worker
	file, err := os.Open(g.fileName)
	if err != nil {
		// Send error for all jobs this worker would have processed
		for job := range jobs {
			results <- FileExtractionResult{
				Index:    job.Index,
				Error:    fmt.Errorf("worker %d failed to open GPK file: %w", workerID, err),
				Filename: job.Entry.Name,
			}
		}
		return
	}
	defer file.Close()
	for job := range jobs {
		ProgressPrintf("    [Worker %d] Extracting %d/%d: %s\n",
			workerID, job.Index+1, job.TotalFiles, job.Entry.Name)

		err := g.extractSingleFile(file, job.Entry, job.OutputDir)
		results <- FileExtractionResult{
			Index:    job.Index,
			Error:    err,
			Filename: job.Entry.Name,
		}
	}
}

// extractSingleFile extracts a single file from the GPK (thread-safe version)
// This function properly handles ComprHeadLen to produce clean, playable OGG files
func (g *GPK) extractSingleFile(file *os.File, entry GPKEntry, outputDir string) error {
	// Use original filename directly (GPK files already have correct extensions)
	outputPath := filepath.Join(outputDir, entry.Name)
	outputDirPath := filepath.Dir(outputPath)

	err := os.MkdirAll(outputDirPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDirPath, err)
	}

	// Seek to the file offset in the GPK archive
	_, err = file.Seek(int64(entry.Header.Offset), 0)
	if err != nil {
		return fmt.Errorf("failed to seek to entry %s: %w", entry.Name, err)
	}

	// Read exactly the compressed length bytes
	fileData := make([]byte, entry.Header.ComprLen)
	_, err = file.Read(fileData)
	if err != nil {
		return fmt.Errorf("failed to read entry %s: %w", entry.Name, err)
	}

	// For OGG files, skip the compression header to get clean OGG data
	var finalData []byte
	if isOGGFile(entry.Name) && entry.Header.ComprHeadLen > 0 {
		if int(entry.Header.ComprHeadLen) < len(fileData) {
			// Skip the compression header bytes
			dataAfterHeader := fileData[entry.Header.ComprHeadLen:]

			// Find the actual start of OGG data by looking for "OggS" signature
			finalData = findOGGStart(dataAfterHeader)
			if finalData == nil {
				// Fallback: if OggS not found, use data after header
				finalData = dataAfterHeader
			}
		} else {
			// Compression header length is invalid, use original data
			finalData = fileData
		}
	} else {
		// Non-OGG files or no compression header, use original data
		finalData = fileData
	}

	return g.writeExtractedFile(outputPath, finalData)
}

// writeExtractedFile writes the processed file data to disk
func (g *GPK) writeExtractedFile(outputPath string, data []byte) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outFile.Close()
	_, err = outFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}
	return nil
}

// isOGGFile checks if a filename represents an OGG audio file
func isOGGFile(filename string) bool {
	return strings.HasSuffix(strings.ToUpper(filename), ".OGG")
}

// findOGGStart finds the actual start of OGG data by looking for "OggS" signature
func findOGGStart(data []byte) []byte {
	// Look for the "OggS" signature in the data
	for i := 0; i <= len(data)-4; i++ {
		if data[i] == 'O' && data[i+1] == 'g' && data[i+2] == 'g' && data[i+3] == 'S' {
			// Found OGG signature, return data starting from here
			return data[i:]
		}
	}
	// OGG signature not found
	return nil
}
