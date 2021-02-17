package test_data

import "embed"

//go:embed hello.txt
var string hello

//go:embed files/*.txt
var txt embed.FS
