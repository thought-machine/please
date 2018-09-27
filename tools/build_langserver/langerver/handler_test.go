package langerver

import (
	"encoding/json"
	"fmt"
	"testing"
	//"encoding/json"
)
type foo struct {
	blah string
	hello string
}

func TestInit(t *testing.T) {
	b := &foo{blah: "hello"}
	h, _ := json.Marshal(b)
	fmt.Println(b.hello == "")
	fmt.Println(string(h))
}