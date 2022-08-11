package test

import (
	"encoding/base64"
	"os"
	"testing"
)

const msg = "UmljY2FyZG8gbGlrZXMgcGluZWFwcGxlIHBpenphCg=="

func TestWriteExtraOutput(t *testing.T) {
	out, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	err = os.WriteFile("truth.txt", out, 0644)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}
