package main

import (
	"log"
	"os"
)

func main() {
	if err := os.WriteFile(os.Getenv("OUT"), []byte(os.Getenv("NAME")), 0600); err != nil {
		log.Fatalf("write '%s': %s", os.Getenv("OUT"), err)
	}
}
