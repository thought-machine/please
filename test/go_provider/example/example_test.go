package example

import (
	"testing"
)

func TestAnswer(t *testing.T) {
	if GetAnswer() != 42 {
		t.Fail()
	}
}
