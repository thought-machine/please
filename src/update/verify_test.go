package update

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// readFile reads a file into an io.Reader
// It uses ioutil.ReadFile because that more closely mimics how we would do this
// for real (i.e. we would read to a buffer over HTTP then verify that, because we
// need to reuse the reader again afterwards and don't want to parse the tarball
// until we're sure it's OK).
func readFile(filename string) io.Reader {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("%s", err)
	}
	return bytes.NewReader(b)
}

func TestVerifyGoodSignature(t *testing.T) {
	signed := readFile("src/update/test_data/test.txt")
	signature := readFile("src/update/test_data/test.txt.asc")
	assert.True(t, verifySignature(signed, signature))
}

func TestVerifyBadSignature(t *testing.T) {
	signed := readFile("src/update/test_data/test.txt")
	signature := readFile("src/update/test_data/bad.txt.asc")
	assert.False(t, verifySignature(signed, signature))
}

func TestVerifyBadFile(t *testing.T) {
	signed := readFile("src/update/test_data/bad.txt")
	signature := readFile("src/update/test_data/test.txt.asc")
	assert.False(t, verifySignature(signed, signature))
}

func TestMustVerifyGoodSignature(t *testing.T) {
	signed := readFile("src/update/test_data/test.txt")
	signature := readFile("src/update/test_data/test.txt.asc")
	r := mustVerifySignature(signed, signature)
	b, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte("Test file for verifying release signatures.\n"), b)
}

func TestMustVerifyBadSignature(t *testing.T) {
	signed := readFile("src/update/test_data/test.txt")
	signature := readFile("src/update/test_data/bad.txt.asc")
	assert.Panics(t, func() { mustVerifySignature(signed, signature) })
}

func TestMustVerifyBadFile(t *testing.T) {
	signed := readFile("src/update/test_data/bad.txt")
	signature := readFile("src/update/test_data/test.txt.asc")
	assert.Panics(t, func() { mustVerifySignature(signed, signature) })
}
