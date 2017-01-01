// +build nobootstrap

// Package help prints help messages about parts of plz.
package help

// Help prints help on a particular topic.
// It returns true if the topic is known or false if it isn't.
func Help(topic string) bool {
	return false
}
