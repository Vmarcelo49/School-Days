package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run investigate_gpk.go <gpk_file> <offset> <length>")
		fmt.Println("Example: go run investigate_gpk.go BGM.GPK 5120 200")
		os.Exit(1)
	}

	filename := os.Args[1]
	offset, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fmt.Printf("Invalid offset: %v\n", err)
		os.Exit(1)
	}

	length, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Printf("Invalid length: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Seek to the specified offset
	_, err = file.Seek(offset, 0)
	if err != nil {
		fmt.Printf("Error seeking to offset %d: %v\n", offset, err)
		os.Exit(1)
	}

	// Read the specified amount of data
	buffer := make([]byte, length)
	n, err := file.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Data from %s at offset %d (reading %d bytes):\n", filename, offset, n)
	fmt.Println("Offset  00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F  ASCII")
	fmt.Println("------  -----------------------------------------------  ----------------")

	for i := 0; i < n; i += 16 {
		// Print offset
		fmt.Printf("%06X  ", offset+int64(i))

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

	// Look for "OggS" patterns in the buffer
	fmt.Printf("\nSearching for 'OggS' pattern in the data:\n")
	for i := 0; i < n-3; i++ {
		if buffer[i] == 'O' && buffer[i+1] == 'g' && buffer[i+2] == 'g' && buffer[i+3] == 'S' {
			fmt.Printf("Found 'OggS' at relative offset %d (absolute offset %d)\n", i, offset+int64(i))
		}
	}
}
