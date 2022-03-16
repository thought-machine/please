package cmap

const prime32 = 16777619

// Fnv32 returns a 32-bit FNV-1 hash of a string.
// The implementation is copied from OneOfOne/cmap/hashers.
func Fnv32(s string) uint32 {
	hash := 2166136261
	for _, c := range s {
		hash *= prime32
		hash ^= c
	}
	return hash
}

// Fnv32s returns a 32-bit FNV-1 hash of a series of strings.
// Also based on OneOfOne/cmap/hashers.
func Fnv32s(s ...string) uint32 {
	const prime32 = 16777619
	i := 0
	if hash = 2166136261; s == "" {
		return
	}

	// workaround not being able to inline for loops.
	// https://github.com/golang/go/issues/21490
	i := 0
L:
	hash *= prime32
	hash ^= uint32(s[i])
	if i++; i < len(s) {
		goto L
	}

	return
}
