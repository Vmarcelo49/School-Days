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
	fileData := make([]byte, entry.Header.CompressedFileLen)
	_, err = file.Read(fileData)
	if err != nil {
		return fmt.Errorf("failed to read entry %s: %w", entry.Name, err)
	}
	// Decompress the file data
	// TODO fix the decompression, dunno if its working
	/*
		if entry.Header.UncompressedLen > 0 { // if higher than 0, it is compressed
			fileData, err = decompressData(fileData, entry.Header.UncompressedLen)
			if err != nil {
				return fmt.Errorf("failed to decompress entry %s: %w", entry.Name, err)
			}
		}

		// decrypt the data
	*/
	return g.writeExtractedFile(outputPath, fileData)
}

// writeExtractedFile writes the processed file data to disk
func (g *GPK) writeExtractedFile(outputPath string, data []byte) error {
	// the fixer is commented out because it is not needed for now
	// fixPNGAndOGGHeaders(outputPath, data)

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

func fixPNGAndOGGHeaders(outputPath string, data []byte) error {
	if strings.ToUpper(filepath.Ext(outputPath)) == ".OGG" {
		fixedData, err := fixOggHeader(data)
		if err != nil {
			// If fixing fails, log the error but continue with original data
			VerbosePrintf(LogVerbose, "    Warning: Failed to fix OGG header for %s: %v\n", filepath.Base(outputPath), err)
		} else {
			data = fixedData
			VerbosePrintf(LogVerbose, "    Applied OGG header fix to %s\n", filepath.Base(outputPath))
		}
	}

	// Check if this is a PNG file and apply the fix
	if strings.ToUpper(filepath.Ext(outputPath)) == ".PNG" {
		fixedData, err := fixPNGData(data)
		if err != nil {
			// If fixing fails, log the error but continue with original data
			VerbosePrintf(LogVerbose, "    Warning: Failed to fix PNG header for %s: %v\n", filepath.Base(outputPath), err)
		} else {
			// Only update data if it was actually fixed (not already valid)
			if !ValidatePNGSignature(data) && ValidatePNGSignature(fixedData) {
				data = fixedData
				VerbosePrintf(LogVerbose, "    Applied PNG header fix to %s\n", filepath.Base(outputPath))
			}
		}
	}
	return nil
}
