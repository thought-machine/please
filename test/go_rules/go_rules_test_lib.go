// Intermediate library used in this test.
package parse

import "test/go_rules/test"

func GetAnswer() int {
	return test.GetAnswer()
}

//go:generate stringer -type=Cat
type Cat int

const (
	Ginger Cat = iota
	Tortoiseshell
	Bengal
	Halp
)
