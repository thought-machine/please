package assets

import (
	// need to be imported to trigger go embed
	_ "embed"
)

// Pleasew is the please wrapper script
//
//go:embed pleasew
var Pleasew []byte

// PleasewPS1 is the Windows counterpart of the wrapper script
//
//go:embed pleasew.ps1
var PleasewPS1 []byte

// PlzComplete is the plz completion script
//
//go:embed plz_complete.sh
var PlzComplete []byte
