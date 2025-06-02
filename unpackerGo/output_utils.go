// Output Control Utilities - provides controlled output based on verbosity settings
package main

import (
	"fmt"
	"os"
)

// LogLevel represents different levels of logging output
type LogLevel int

const (
	LogQuiet   LogLevel = iota // Only errors and essential output
	LogNormal                  // Standard output
	LogVerbose                 // Detailed output
	LogDebug                   // All debug information
)

// Printf prints formatted output only if the current verbosity level allows it
func VerbosePrintf(level LogLevel, format string, args ...interface{}) {
	if shouldPrint(level) {
		fmt.Printf(format, args...)
	}
}

// Println prints a line only if the current verbosity level allows it
func VerbosePrintln(level LogLevel, args ...interface{}) {
	if shouldPrint(level) {
		fmt.Println(args...)
	}
}

// Print prints without newline only if the current verbosity level allows it
func VerbosePrint(level LogLevel, args ...interface{}) {
	if shouldPrint(level) {
		fmt.Print(args...)
	}
}

// ErrorPrintf always prints error messages to stderr regardless of verbosity
func ErrorPrintf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// InfoPrintf prints normal information (respects quiet mode)
func InfoPrintf(format string, args ...interface{}) {
	VerbosePrintf(LogNormal, format, args...)
}

// DebugPrintf prints debug information only in debug or verbose mode
func DebugPrintf(format string, args ...interface{}) {
	VerbosePrintf(LogDebug, format, args...)
}

// shouldPrint determines if output should be printed based on current settings and level
func shouldPrint(level LogLevel) bool {
	if IsQuietMode && level > LogQuiet {
		return false
	}

	if IsDebugMode {
		return true // Debug mode shows everything
	}

	if IsVerboseMode && level <= LogVerbose {
		return true
	}

	if !IsVerboseMode && !IsQuietMode && level <= LogNormal {
		return true
	}

	return false
}

// ProgressPrintf prints progress information (shows unless in quiet mode)
func ProgressPrintf(format string, args ...interface{}) {
	if !IsQuietMode {
		fmt.Printf(format, args...)
	}
}

// ResultPrintf prints final results (always shows unless in quiet mode)
func ResultPrintf(format string, args ...interface{}) {
	if !IsQuietMode {
		fmt.Printf(format, args...)
	}
}
