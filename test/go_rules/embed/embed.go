package embed

import "embed"

//go:embed hello.txt
var hello string

//go:embed test_data
var testData embed.FS
