// GPK Unpacker - Command Line Interface
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CLIConfig holds all command line configuration
type CLIConfig struct {
	DebugMode   bool
	VerboseMode bool
	QuietMode   bool
	OutputDir   string
	FixPngMode  bool
	DecryptOnly bool
	InputPath   string
}

// Global debug control variables
var (
	IsVerboseMode = false
	IsQuietMode   = false
	IsDebugMode   = false
)

// parseCommandLine handles all command line parsing and flag setup
func parseCommandLine() *CLIConfig {
	// Define command line flags
	var config CLIConfig
	flag.BoolVar(&config.DebugMode, "debug", false, "Show compression information for GPK file")
	flag.BoolVar(&config.VerboseMode, "verbose", false, "Enable verbose output and detailed processing information")
	flag.BoolVar(&config.VerboseMode, "v", false, "Enable verbose output (short form)")
	flag.BoolVar(&config.QuietMode, "quiet", false, "Suppress all non-essential output")
	flag.BoolVar(&config.QuietMode, "q", false, "Suppress all non-essential output (short form)")
	flag.StringVar(&config.OutputDir, "output", "extracted", "Output directory for extracted files")
	flag.BoolVar(&config.FixPngMode, "fix-png", false, "Fix corrupted PNG files in specified directory")
	flag.BoolVar(&config.DecryptOnly, "decryptOnly", false, "Decrypt GPK file without extracting contents")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "GPK Batch Unpacker\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s [flags] <gpk_file_or_directory>\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nSpecial Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check-headers         Check compression headers\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s BGM.GPK\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -output extracted \"D:\\Games\\Overflow\\SCHOOLDAYS HQ\\Packs\"\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -debug BGM.GPK\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -verbose BGM.GPK\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -quiet BGM.GPK\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -fix-png extracted/\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -decryptOnly source.gpk\n", os.Args[0])
	}
	flag.Parse()

	// Set global debug control variables
	IsVerboseMode = config.VerboseMode
	IsQuietMode = config.QuietMode
	IsDebugMode = config.DebugMode

	// Validate flags - can't be both verbose and quiet
	if config.VerboseMode && config.QuietMode {
		fmt.Fprintf(os.Stderr, "Error: Cannot use both -verbose and -quiet flags simultaneously\n")
		os.Exit(1)
	}

	// Check if we have the required positional argument
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	config.InputPath = flag.Arg(0)
	return &config
}

// runCLI executes the main CLI logic based on parsed configuration
func runCLI(config *CLIConfig) error {
	// PNG fix mode - fix corrupted PNG files in directory
	if config.FixPngMode {
		return FixAllPNGFiles(config.InputPath)
	}

	// Decrypt only mode - decrypt GPK file without extraction
	if config.DecryptOnly {
		return decryptGPKFile(config.InputPath)
	}

	// Debug mode - show compression information
	if config.DebugMode {
		debugCompressionInfo(config.InputPath)
		return nil
	}

	// Check if input is a file or directory
	stat, err := os.Stat(config.InputPath)
	if err != nil {
		return fmt.Errorf("error accessing path %s: %w", config.InputPath, err)
	}
	if stat.IsDir() {
		// Batch mode - process all GPK files in directory
		InfoPrintf("Batch mode: Processing all GPK files in: %s\n", config.InputPath)
		return processBatch(config.InputPath, config.OutputDir)
	} else {
		// Single file mode
		InfoPrintf("Single file mode: Processing: %s\n", config.InputPath)
		return processSingleFile(config.InputPath, config.OutputDir)
	}
}

// debugCompressionInfo shows compression information for GPK entries
func debugCompressionInfo(gpkFile string) {
	gpk := NewGPK()
	err := gpk.Load(gpkFile)
	if err != nil {
		ErrorPrintf("Failed to load GPK: %v\n", err)
		os.Exit(1)
	}

	InfoPrintf("GPK file: %s\n", gpkFile)
	InfoPrintf("Total entries: %d\n\n", len(gpk.GetEntries()))

	// Show compression info for first 10 entries
	InfoPrintf("Compression information for first 10 entries:\n")
	InfoPrintf("Name\t\t\t\tOffset\t\tComprLen\tUncomprLen\tComprHeadLen\n")
	InfoPrintf("================================================================================\n")

	for i, entry := range gpk.GetEntries() {
		if i >= 10 {
			break
		}
		InfoPrintf("%-30s\t%d\t\t%d\t\t%d\t\t%d\n",
			entry.Name,
			entry.Header.Offset,
			entry.Header.ComprLen,
			entry.Header.UncomprLen,
			entry.Header.ComprHeadLen)
	}
}

// decryptGPKFile decrypts a GPK file following the rules in GPK_Decrypted_Structure_Documentation.md
// Only the PIDX section and signature section are decrypted, not the file data
func decryptGPKFile(inputPath string) error {
	// Validate input is a GPK file
	if !strings.HasSuffix(strings.ToLower(inputPath), ".gpk") {
		return fmt.Errorf("input file must be a .gpk file")
	}

	// Check if file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", inputPath)
	}

	InfoPrintf("Decrypting GPK file: %s\n", inputPath)

	// Open source file
	sourceFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get file info for size
	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Generate output filename
	dir := filepath.Dir(inputPath)
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(dir, baseName+"_decrypted.gpk")

	InfoPrintf("Output file: %s\n", outputPath)

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// First, read and decrypt the signature to get PIDX length
	const signatureSize = 32
	if fileSize < signatureSize {
		return fmt.Errorf("file too small to contain GPK signature")
	}

	// Read the encrypted signature (last 32 bytes)
	_, err = sourceFile.Seek(fileSize-signatureSize, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to signature: %w", err)
	}

	encryptedSig := make([]byte, signatureSize)
	_, err = sourceFile.Read(encryptedSig)
	if err != nil {
		return fmt.Errorf("failed to read encrypted signature: %w", err)
	}

	// Decrypt the signature to read PIDX length
	decryptedSig := make([]byte, signatureSize)
	copy(decryptedSig, encryptedSig)
	decryptData(decryptedSig) // Parse the signature to get PIDX length
	signature, err := readGPKSignature(bytes.NewReader(decryptedSig))
	if err != nil {
		return fmt.Errorf("failed to parse decrypted signature: %w", err)
	}

	// Verify it's a valid GPK signature - try decrypted first, then original if needed
	isValidDecrypted := string(signature.Sig0[:len(GPKTailerIdent0)]) == GPKTailerIdent0 &&
		string(signature.Sig1[:len(GPKTailerIdent1)]) == GPKTailerIdent1

	if !isValidDecrypted {
		// Try without decryption - some GPK files may have unencrypted signatures
		VerbosePrintf(LogVerbose, "Decrypted signature invalid, trying original signature...\n")
		signatureOriginal, err := readGPKSignature(bytes.NewReader(encryptedSig))
		if err != nil {
			return fmt.Errorf("failed to parse original signature: %w", err)
		}

		isValidOriginal := string(signatureOriginal.Sig0[:len(GPKTailerIdent0)]) == GPKTailerIdent0 &&
			string(signatureOriginal.Sig1[:len(GPKTailerIdent1)]) == GPKTailerIdent1

		if isValidOriginal {
			VerbosePrintf(LogVerbose, "Using original unencrypted signature\n")
			signature = signatureOriginal
			copy(decryptedSig, encryptedSig) // Use the original signature for output
		} else {
			return fmt.Errorf("invalid GPK signature - neither encrypted nor decrypted version is valid")
		}
	} else {
		VerbosePrintf(LogVerbose, "Using decrypted signature\n")
	}

	// Calculate section boundaries
	pidxOffset := fileSize - signatureSize - int64(signature.PidxLength)
	if pidxOffset < 0 {
		return fmt.Errorf("invalid PIDX length: %d (file size: %d)", signature.PidxLength, fileSize)
	}

	InfoPrintf("File structure detected:\n")
	InfoPrintf("  File data: 0 to %d (%d bytes) - will remain unencrypted\n", pidxOffset-1, pidxOffset)
	InfoPrintf("  PIDX section: %d to %d (%d bytes) - will be decrypted\n", pidxOffset, fileSize-signatureSize-1, signature.PidxLength)
	InfoPrintf("  Signature: %d to %d (%d bytes) - will be decrypted\n", fileSize-signatureSize, fileSize-1, signatureSize)

	// Reset to beginning of file
	_, err = sourceFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to beginning: %w", err)
	}

	// Copy file data section unchanged (unencrypted)
	if !IsQuietMode {
		fmt.Printf("Copying file data section...")
	}

	const chunkSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, chunkSize)
	var totalProcessed int64

	// Copy file data section without decryption
	for totalProcessed < pidxOffset {
		remaining := pidxOffset - totalProcessed
		readSize := chunkSize
		if remaining < int64(readSize) {
			readSize = int(remaining)
		}

		n, err := sourceFile.Read(buffer[:readSize])
		if n == 0 || (err != nil && err != io.EOF) {
			return fmt.Errorf("failed to read file data: %w", err)
		}

		// Write file data unchanged (no decryption)
		_, writeErr := outputFile.Write(buffer[:n])
		if writeErr != nil {
			return fmt.Errorf("failed to write file data: %w", writeErr)
		}

		totalProcessed += int64(n)

		if !IsQuietMode {
			progress := float64(totalProcessed) / float64(pidxOffset) * 100
			fmt.Printf("\rCopying file data: %.1f%% (%d/%d bytes)", progress, totalProcessed, pidxOffset)
		}
	}

	if !IsQuietMode {
		fmt.Println() // New line
		fmt.Printf("Decrypting PIDX section...")
	}

	// Read and decrypt the PIDX section
	pidxData := make([]byte, signature.PidxLength)
	_, err = sourceFile.Read(pidxData)
	if err != nil {
		return fmt.Errorf("failed to read PIDX data: %w", err)
	}

	// Decrypt PIDX data
	decryptData(pidxData)

	// Write decrypted PIDX section
	_, err = outputFile.Write(pidxData)
	if err != nil {
		return fmt.Errorf("failed to write decrypted PIDX: %w", err)
	}

	if !IsQuietMode {
		fmt.Printf(" done (%d bytes)\n", len(pidxData))
		fmt.Printf("Writing decrypted signature...")
	}

	// Write the decrypted signature
	_, err = outputFile.Write(decryptedSig)
	if err != nil {
		return fmt.Errorf("failed to write decrypted signature: %w", err)
	}

	if !IsQuietMode {
		fmt.Printf(" done (%d bytes)\n", len(decryptedSig))
	}
	ResultPrintf("Successfully created properly decrypted GPK file: %s\n", outputPath)
	ResultPrintf("File data section: %d bytes (preserved unencrypted)\n", pidxOffset)
	ResultPrintf("PIDX section: %d bytes (decrypted)\n", signature.PidxLength)
	ResultPrintf("Signature section: %d bytes (decrypted)\n", signatureSize)
	ResultPrintf("Total file size: %d bytes\n", fileSize)

	// Show debug information about the decrypted entries
	if IsVerboseMode || IsDebugMode {
		InfoPrintf("\n=== DECRYPTED GPK ENTRY INFORMATION ===\n")
		err = showDecryptedGPKInfo(outputPath)
		if err != nil {
			VerbosePrintf(LogVerbose, "Warning: Could not show decrypted GPK info: %v\n", err)
		}
	}

	return nil
}

// showDecryptedGPKInfo displays detailed information about entries in a decrypted GPK file
func showDecryptedGPKInfo(gpkFile string) error {
	gpk := NewGPK()
	err := gpk.Load(gpkFile)
	if err != nil {
		return fmt.Errorf("failed to load decrypted GPK: %w", err)
	}

	InfoPrintf("Decrypted GPK file: %s\n", gpkFile)
	InfoPrintf("Total entries: %d\n\n", len(gpk.GetEntries()))

	// Show information for first 10 entries
	InfoPrintf("Entry information (first 10 entries):\n")
	InfoPrintf("%-30s %-10s %-10s %-12s %-12s\n", "Name", "Offset", "ComprLen", "UncomprLen", "ComprHeadLen")
	InfoPrintf("================================================================================\n")

	for i, entry := range gpk.GetEntries() {
		if i >= 10 {
			break
		}
		InfoPrintf("%-30s %-10d %-10d %-12d %-12d\n",
			entry.Name,
			entry.Header.Offset,
			entry.Header.ComprLen,
			entry.Header.UncomprLen,
			entry.Header.ComprHeadLen)
	}

	// If there are OGG files, show more detailed analysis
	oggCount := 0
	for _, entry := range gpk.GetEntries() {
		if strings.HasSuffix(strings.ToUpper(entry.Name), ".OGG") {
			oggCount++
		}
	}

	if oggCount > 0 {
		InfoPrintf("\nFound %d OGG files in the archive\n", oggCount)

		if IsVerboseMode {
			InfoPrintf("\nDetailed OGG file analysis (first 5 OGG files):\n")
			InfoPrintf("================================================================================\n")

			oggProcessed := 0
			file, err := os.Open(gpkFile)
			if err != nil {
				return fmt.Errorf("failed to open GPK file for analysis: %w", err)
			}
			defer file.Close()

			for _, entry := range gpk.GetEntries() {
				if strings.HasSuffix(strings.ToUpper(entry.Name), ".OGG") && oggProcessed < 5 {
					InfoPrintf("\nFile: %s\n", entry.Name)
					InfoPrintf("  Offset: %d, Size: %d bytes\n", entry.Header.Offset, entry.Header.ComprLen)

					// Check the first few bytes at the offset to see if they contain "OggS"
					file.Seek(int64(entry.Header.Offset), 0)
					headerBytes := make([]byte, 32)
					n, _ := file.Read(headerBytes)

					if n > 0 {
						InfoPrintf("  First 32 bytes: ")
						for i := 0; i < n; i++ {
							fmt.Printf("%02X ", headerBytes[i])
						}
						fmt.Printf("\n")

						// Check for OggS signature
						oggSFound := false
						for i := 0; i <= n-4; i++ {
							if string(headerBytes[i:i+4]) == "OggS" {
								InfoPrintf("  ✓ Found 'OggS' signature at position %d\n", i)
								oggSFound = true
								break
							}
						}
						if !oggSFound {
							InfoPrintf("  ⚠ No 'OggS' signature found in first 32 bytes\n")
						}
					}

					oggProcessed++
				}
			}
		}
	}

	return nil
}
