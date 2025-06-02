// GPK Batch Unpacker - Main Entry Point
package main

import (
	"os"
)

func main() {
	// Parse command line arguments and configuration
	config := parseCommandLine()
	// Run the main CLI logic
	err := runCLI(config)
	if err != nil {
		ErrorPrintf("Error: %v\n", err)
		os.Exit(1)
	}

	ResultPrintf("\nOperation completed successfully!\n")
}
