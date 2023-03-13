package assets

import (
	// need to be imported to trigger go embed
	_ "embed"
)

// Pleasew is the please wrapper script
//
//go:embed pleasew
var Pleasew []byte

// PlzComplete is the plz completion script
//
//go:embed plz_complete.sh
var PlzComplete []byte
