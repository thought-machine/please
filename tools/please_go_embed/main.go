// Package main implements interpretation of //go:embed directives to convert them into an 'embedcfg' JSON struct.
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/thought-machine/please/tools/please_go_embed/embed"
)

func main() {
	if err := parseEmbeds(os.Args[1:]); err != nil {
		log.Fatalf("Failed to embed files: %s", err)
	}
}

func parseEmbeds(filenames []string) error {
	cfg, err := embed.Parse(filenames)
	if err != nil {
		return err
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	os.Stdout.Write(b)
	return nil
}
