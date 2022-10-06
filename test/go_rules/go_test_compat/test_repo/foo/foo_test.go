package foo

import (
	"os"
	"testing"
)

func TestFoo(t *testing.T) {
	from, err := os.ReadFile("test_data/bar.txt")
	if err != nil {
		panic(err)
	}

	println("writing " + string(from))

	err = os.WriteFile("bar.txt", from, 0444)
	if err != nil {
		panic(err)
	}
}
