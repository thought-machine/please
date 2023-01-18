package rules

import "embed"

//go:embed *.build_defs
var builtins embed.FS

// AllAssets returns all assets embedded into the binary
func AllAssets(excludes map[string]struct{}) ([]string, error) {
	assets, err := builtins.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var filepaths []string
	for _, entry := range assets {
		if _, ok := excludes[entry.Name()]; !ok {
			filepaths = append(filepaths, entry.Name())
		}
	}

	return filepaths, nil
}

// ReadAsset reads the contents of a specified build definition asset. If the specified
// file does not exist an error is returned.
func ReadAsset(path string) ([]byte, error) {
	return builtins.ReadFile(path)
}
