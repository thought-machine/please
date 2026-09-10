package core

import "strings"

// normalisePathSeparators rewrites the paths Please generates to use forward slashes.
//
// Build commands are shell strings, and a backslash is an escape character to much of what
// runs in them. Expanding a variable is safe, but passing one to anything that interprets its
// arguments is not: `sed -e "s#x#$TMP_DIR#"` silently turns `\tmp` into a tab, and the C/C++
// rules build their link line with sed. Win32, MinGW and busybox all accept forward slashes,
// so we use those throughout.
//
// This deliberately runs before withUserProvidedEnv: values the user wrote themselves are
// left exactly as written, since they may not be paths at all.
func (env BuildEnv) normalisePathSeparators() {
	for k, v := range env {
		if strings.ContainsRune(v, '\\') {
			env[k] = strings.ReplaceAll(v, `\`, `/`)
		}
	}
}
