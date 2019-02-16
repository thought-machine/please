// +build bootstrap

// Package bazel provides stubs for Bazel extensions for Please.
package bazel

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
