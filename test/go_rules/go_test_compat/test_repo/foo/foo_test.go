package foo

import (
	"io/ioutil"
	"testing"
)

func TestFoo(t *testing.T) {
	from, err := ioutil.ReadFile("test_data/bar.txt")
	if err != nil {
		panic(err)
	}

	println("writing " + string(from))

	err = ioutil.WriteFile("bar.txt", from, 0444)
	if err != nil {
		panic(err)
	}
}