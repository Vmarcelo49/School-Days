package main

import (
	"log"

	"school-days-engine/internal/engine"
)

func main() {
	game := engine.NewGame()
	if err := game.Run(); err != nil {
		log.Fatal(err)
	}
}
