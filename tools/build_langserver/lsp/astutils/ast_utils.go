package astutils

import "strings"

func TrimStrLit(lit string) string {
	return strings.Trim(strings.TrimLeft(lit, "fr"), "\"'")
}
