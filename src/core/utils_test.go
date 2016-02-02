package core

import (
	"crypto/sha1"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseHash(t *testing.T) {
	// Test that these two come out differently
	input1 := [sha1.Size * 4]byte{}
	input2 := [sha1.Size * 4]byte{}
	for i := 0; i < sha1.Size; i++ {
		input1[i] = byte(i)
		input2[i] = byte(i * 2)
	}
	output1 := CollapseHash(input1[:])
	output2 := CollapseHash(input2[:])
	assert.NotEqual(t, output1, output2)
}

func TestCollapseHash2(t *testing.T) {
	// Test of a couple of cases that weren't different...
	input1, err1 := base64.URLEncoding.DecodeString("mByUsoTswXV2X_W6FHhBwJUCQM-YHJSyhOzBdXZf9boUeEHAlQJAz-DzaA7MCXxt5_FFws2WO51vKlqt-JThKzdEQn_bghpDDCuKOI9qGNI=")
	input2, err2 := base64.URLEncoding.DecodeString("rSH0PS_dftB6KN_Jnu_jszhbxiutIfQ9L91-0Hoo38me7-OzOFvGK-DzaA7MCXxt5_FFws2WO51vKlqt-JThKzdEQn_bghpDDCuKOI9qGNI=")
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	output1 := CollapseHash(input1)
	output2 := CollapseHash(input2)
	assert.NotEqual(t, output1, output2)
}
