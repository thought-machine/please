// Package help prints help messages about parts of plz.
package help

// Help is a stub implementation used only during bootstrap, when
// the main help package isn't available.
func Help(topic string) bool {
	return false
}

// Topics is also a stub implementation used only during bootstrap.
func Topics(prefix string) {}

// A Topic is an alias for a string, which does not provide completion during bootstrap.
type Topic string
