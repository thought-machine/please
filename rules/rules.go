package rules

import "embed"

//go:embed *.build_defs
var builtins embed.FS

// AllAssets returns all assets embedded into the binary
func AllAssets() ([]string, error) {
	assets, err := builtins.ReadDir(".")
	if err != nil {
		return nil, err
	}

	filepaths := make([]string, len(assets))
	for i, entry := range assets {
		filepaths[i] = entry.Name()
	}

	return filepaths, nil
}

// ReadAsset reads the contents of a specified build definition asset. If the specified
// file does not exist an error is returned.
func ReadAsset(path string) ([]byte, error) {
	return builtins.ReadFile(path)
}
