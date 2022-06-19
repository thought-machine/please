package testdata

import "embed"

//go:embed hello.txt
var hello string

//go:embed files/*.txt
var txt embed.FS

//go:embed files
var dir embed.FS
