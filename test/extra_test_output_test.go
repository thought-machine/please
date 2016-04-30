package test

import (
	"encoding/base64"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

const msg = "UmljY2FyZG8gbGlrZXMgcGluZWFwcGxlIHBpenphCg=="

func TestWriteExtraOutput(t *testing.T) {
	out, err := base64.StdEncoding.DecodeString(msg)
	assert.NoError(t, err)
	err = ioutil.WriteFile("truth.txt", out, 0644)
	assert.NoError(t, err)
}
