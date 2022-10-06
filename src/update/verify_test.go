package update

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// readFile reads a file into an io.Reader
// It uses os.ReadFile because that more closely mimics how we would do this
// for real (i.e. we would read to a buffer over HTTP then verify that, because we
// need to reuse the reader again afterwards and don't want to parse the tarball
// until we're sure it's OK).
func readFile(filename string) io.Reader {
	b, err := os.ReadFile(filename)
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
	r := mustVerifySignature(signed, signature, true)
	b, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte("Test file for verifying release signatures.\n"), b)
}

func TestMustVerifyBadSignature(t *testing.T) {
	signed := readFile("src/update/test_data/test.txt")
	signature := readFile("src/update/test_data/bad.txt.asc")
	assert.Panics(t, func() { mustVerifySignature(signed, signature, true) })
}

func TestMustVerifyBadFile(t *testing.T) {
	signed := readFile("src/update/test_data/bad.txt")
	signature := readFile("src/update/test_data/test.txt.asc")
	assert.Panics(t, func() { mustVerifySignature(signed, signature, true) })
}

func TestMustVerifyHash(t *testing.T) {
	r := readFile("src/update/test_data/test.txt")
	r = mustVerifyHash(r, []string{
		"d5ddcfb56bee0bf465da6d8e0ab0db5b4635061b45be18c231a558cf1d86c2e0",
	})
	b, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte("Test file for verifying release signatures.\n"), b)
}

func TestMustVerifyHashMultiple(t *testing.T) {
	r := readFile("src/update/test_data/test.txt")
	mustVerifyHash(r, []string{
		"510dc30e9c55d5da05d971bed8568534667640b70295f78082967207745afec0",
		"d5ddcfb56bee0bf465da6d8e0ab0db5b4635061b45be18c231a558cf1d86c2e0",
	})
}

func TestMustVerifyHashBad(t *testing.T) {
	r := readFile("src/update/test_data/test.txt")
	assert.Panics(t, func() {
		mustVerifyHash(r, []string{
			"510dc30e9c55d5da05d971bed8568534667640b70295f78082967207745afec0",
			"877265724bc9ba415dc1774569384fcfde80c6694d70f8f29405a917bbdf09db",
		})
	})
}
