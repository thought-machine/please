package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/please-build/gcfg"

	"github.com/thought-machine/please/src/core"
)

// Config prints configuration settings in human-readable format.
func Config(config *core.Configuration, options []string) {
	if len(options) == 0 {
		v, err := gcfg.Stringify(config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print(v)
	} else {
		for _, option := range options {
			section, subsection, name, err := parseOption(option)
			if err != nil {
				log.Fatal(err)
			}

			values, err := gcfg.Get(config, section, subsection, name)
			if err != nil {
				log.Fatalf("Failed to get %s: %s", option, err)
			}

			for _, value := range values {
				fmt.Println(value)
			}
		}
	}
}

// ConfigJSON prints the configuration settings as JSON.
func ConfigJSON(config *core.Configuration) {
	data, err := gcfg.RawJSON(config)
	if err != nil {
		log.Fatalf("Failed to get JSON configuration: %s", err)
	}

	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "    "); err != nil {
		log.Fatalf("Failed to parse JSON configuration: %s", err)
	}

	fmt.Print(out.String())
}

func parseOption(option string) (section, subsection, name string, err error) {
	parts := strings.Split(option, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", "", fmt.Errorf("Bad option format. Example: section.subsection.name or section.name")
	}
	if len(parts) == 2 {
		return parts[0], "", parts[1], nil
	}
	return parts[0], parts[1], parts[2], nil
}
