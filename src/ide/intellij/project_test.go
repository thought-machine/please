package intellij

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNewMisc(t *testing.T) {
	misc := NewMisc(7)

	buf := &bytes.Buffer{}
	misc.toXml(buf)
	fmt.Println(buf.String())
}