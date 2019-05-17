// +build bootstrap

// Package rules provides stubs for the builtin rules for Please.
package rules

import "fmt"

// AssetDir returns all the builtin files.
func AssetDir(name string) ([]string, error) {
	return nil, nil
}

// Asset returns a builtin file.
func Asset(name string) ([]byte, error) {
	return nil, fmt.Errorf("not found")
}

// MustAsset panics when it can't find a builtin file.
func MustAsset(name string) []byte {
	panic("not found")
}
