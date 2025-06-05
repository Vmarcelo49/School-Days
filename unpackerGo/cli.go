package main

import (
	"flag"
	"fmt"
	"os"
)

// CLIConfig holds all command line configuration
type CLIConfig struct {
	DebugMode   bool
	VerboseMode bool
	QuietMode   bool
	OutputDir   string
	AudioPlayer bool
	InputPath   string
}

// Global debug control variables
var (
	IsVerboseMode = false
	IsQuietMode   = false
	IsDebugMode   = false
)

// parseCommandLine handles all command line parsing and flag setup
func parseCommandLine() *CLIConfig { // Define command line flags
	var config CLIConfig
	flag.BoolVar(&config.DebugMode, "debug", false, "Show compression information for GPK file")
	flag.BoolVar(&config.VerboseMode, "verbose", false, "Enable verbose output and detailed processing information")
	flag.BoolVar(&config.VerboseMode, "v", false, "Enable verbose output (short form)")
	flag.BoolVar(&config.QuietMode, "quiet", false, "Suppress all non-essential output")
	flag.BoolVar(&config.QuietMode, "q", false, "Suppress all non-essential output (short form)")
	flag.StringVar(&config.OutputDir, "output", "extracted", "Output directory for extracted files")
	flag.BoolVar(&config.AudioPlayer, "audio-player", false, "Launch the interactive audio player GUI")

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
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -audio-player\n", os.Args[0])
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
	// Check if we have the required positional argument (not needed for audio player)
	if flag.NArg() < 1 && !config.AudioPlayer {
		flag.Usage()
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		config.InputPath = flag.Arg(0)
	}
	return &config
}

// runCLI executes the main CLI logic based on parsed configuration
func runCLI(config *CLIConfig) error {
	// Audio player mode - launch the interactive GUI
	if config.AudioPlayer {
		InfoPrintf("Launching interactive audio player...\n")
		runGameWindow()
		return nil
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
			entry.Header.CompressedFileLen,
			entry.Header.UncompressedLen,
			entry.Header.PidxDataHeaderLen)
	}
}
