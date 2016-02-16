// Used for testing the builtin Go rules.
package main

import "fmt"
import "go_rules/generate_test"

func main() {
	fmt.Println(generate_test.Placebo.String())
}
