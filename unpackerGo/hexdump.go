package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run hexdump.go <file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read first 128 bytes
	buffer := make([]byte, 128)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("First %d bytes of %s:\n", n, filename)
	fmt.Println("Offset  00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F  ASCII")
	fmt.Println("------  -----------------------------------------------  ----------------")

	for i := 0; i < n; i += 16 {
		// Print offset
		fmt.Printf("%06X  ", i)

		// Print hex bytes
		for j := 0; j < 16; j++ {
			if i+j < n {
				fmt.Printf("%02X ", buffer[i+j])
			} else {
				fmt.Print("   ")
			}
		}

		// Print ASCII representation
		fmt.Print(" ")
		for j := 0; j < 16 && i+j < n; j++ {
			b := buffer[i+j]
			if b >= 32 && b <= 126 {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println()
	}

	// Look for "OggS" pattern in the first 128 bytes
	fmt.Printf("\nSearching for 'OggS' pattern:\n")
	for i := 0; i < n-3; i++ {
		if buffer[i] == 'O' && buffer[i+1] == 'g' && buffer[i+2] == 'g' && buffer[i+3] == 'S' {
			fmt.Printf("Found 'OggS' at offset %d (0x%X)\n", i, i)
		}
	}
}
