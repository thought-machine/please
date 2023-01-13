package update

import (
	"bytes"
	"crypto"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/stretchr/testify/assert"
)

var goodKey = loadKey("good.pem")
var badKey = loadKey("bad.pem")

var goodKeyPub = loadKey("good.pub.pem")

func loadKey(name string) []byte {
	key, err := os.ReadFile(filepath.Join("src/update/test_data", name))
	if err != nil {
		panic(err)
	}
	return key
}

func sign(key []byte, msg []byte) io.Reader {
	priv, err := cryptoutils.UnmarshalPEMToPrivateKey(key, cryptoutils.SkipPassword)
	if err != nil {
		panic(err)
	}

	verifier, err := signature.LoadSignerVerifier(priv, crypto.SHA256)
	if err != nil {
		panic(err)
	}

	bs, err := verifier.SignMessage(bytes.NewReader(msg))
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(bs)
}

// readFile reads a file into an io.Reader
// It uses os.ReadFile because that more closely mimics how we would do this
// for real (i.e. we would read to a buffer over HTTP then verify that, because we
// need to reuse the reader again afterwards and don't want to parse the tarball
// until we're sure it's OK).
func readFile(filename string) []byte {
	b, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("%s", err)
	}
	return b
}

func fileReader(filename string) io.Reader {
	return bytes.NewReader(readFile(filename))
}

func TestVerifyGoodSignature(t *testing.T) {
	message := readFile("src/update/test_data/test.txt")
	assert.True(t, verifySignatureWithKey(bytes.NewReader(message), sign(goodKey, message), goodKeyPub))
}

func TestVerifyBadSignature(t *testing.T) {
	message := readFile("src/update/test_data/test.txt")
	assert.False(t, verifySignatureWithKey(bytes.NewReader(message), sign(badKey, message), goodKeyPub))
}

func TestMustVerifyHash(t *testing.T) {
	r := fileReader("src/update/test_data/test.txt")
	r = mustVerifyHash(r, []string{
		"d5ddcfb56bee0bf465da6d8e0ab0db5b4635061b45be18c231a558cf1d86c2e0",
	})
	b, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte("Test file for verifying release signatures.\n"), b)
}

func TestMustVerifyHashMultiple(t *testing.T) {
	r := fileReader("src/update/test_data/test.txt")
	mustVerifyHash(r, []string{
		"510dc30e9c55d5da05d971bed8568534667640b70295f78082967207745afec0",
		"d5ddcfb56bee0bf465da6d8e0ab0db5b4635061b45be18c231a558cf1d86c2e0",
	})
}

func TestMustVerifyHashBad(t *testing.T) {
	r := fileReader("src/update/test_data/test.txt")
	assert.Panics(t, func() {
		mustVerifyHash(r, []string{
			"510dc30e9c55d5da05d971bed8568534667640b70295f78082967207745afec0",
			"877265724bc9ba415dc1774569384fcfde80c6694d70f8f29405a917bbdf09db",
		})
	})
}
