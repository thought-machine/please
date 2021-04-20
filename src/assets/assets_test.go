package assets_test

import (
	"testing"

	"github.com/thought-machine/please/src/assets"
)

func TestAssets(t *testing.T) {
	if len(assets.Pleasew) == 0 && string(assets.Pleasew) != "dummy" {
		panic("Pleasew was not set via embed")
	}
	if len(assets.PlzComplete) == 0 && string(assets.PlzComplete) != "dummy" {
		panic("PlzComplete was not set via embed")
	}
}
