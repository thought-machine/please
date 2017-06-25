// +build nobootstrap

package utils

import (
	"fmt"
)

// PrintCompletionScript prints Please's completion script to stdout.
func PrintCompletionScript() {
	fmt.Printf("%s\n", MustAsset("plz_complete.sh"))
}
