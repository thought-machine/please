// +build nobootstrap

package utils

import (
	"fmt"
)

// PrintCompletionScript prints Please's completion script to stdout.
func PrintCompletionScript(zsh bool) {
	if zsh {
		fmt.Printf("%s\n", MustAsset("plz_complete.zsh"))
	} else {
		fmt.Printf("%s\n", MustAsset("plz_complete.sh"))
	}
}
