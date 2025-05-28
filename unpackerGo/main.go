package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	gameRoot := "/media/work/Dev/Games/SD"

	// Allow override from command line argument
	if len(os.Args) > 1 {
		gameRoot = os.Args[1]
	}

	fmt.Printf("Unpacking from: %s\n", gameRoot)

	fs, err := NewFileSystem(gameRoot)
	if err != nil {
		log.Fatalf("Failed to initialize filesystem: %v", err)
	}

	err = fs.UnpackAll()
	if err != nil {
		log.Fatalf("Failed to unpack: %v", err)
	}

	fmt.Println("Unpacking completed successfully!")
}
