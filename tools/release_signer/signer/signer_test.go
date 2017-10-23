package signer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/openpgp"
)

const (
	pubKey  = "tools/release_signer/signer/test_data/pub.gpg"
	secKey  = "tools/release_signer/signer/test_data/sec.gpg"
	testTxt = "tools/release_signer/signer/test_data/test.txt"
	badTxt  = "tools/release_signer/signer/test_data/bad.txt"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func verifyFile(signed, signature, keyring string) bool {
	f1, err := os.Open(signed)
	must(err)
	f2, err := os.Open(signature)
	must(err)
	f3, err := os.Open(keyring)
	must(err)
	entities, err := openpgp.ReadArmoredKeyRing(f3)
	must(err)
	_, err = openpgp.CheckArmoredDetachedSignature(entities, f1, f2)
	return err == nil
}

func TestSignFile(t *testing.T) {
	assert.NoError(t, SignFile(testTxt, "test.txt.asc", secKey, "test@please.build", "testtest"))
	assert.True(t, verifyFile(testTxt, "test.txt.asc", pubKey))
}

func TestSignFileBadPassphrase(t *testing.T) {
	assert.Error(t, SignFile(testTxt, "test.txt.asc", secKey, "test@please.build", "nope"))
}

func TestSignFileBadSignature(t *testing.T) {
	assert.NoError(t, SignFile(testTxt, "test.txt.asc", secKey, "test@please.build", "testtest"))
	assert.False(t, verifyFile(badTxt, "test.txt.asc", pubKey))
}

func TestSignFileUnknownUser(t *testing.T) {
	assert.Error(t, SignFile(testTxt, "test.txt.asc", secKey, "not@please.build", "testtest"))
}

func TestSignFileMissingKeyring(t *testing.T) {
	assert.Error(t, SignFile(testTxt, "test.txt.asc", "doesnt_exist", "test@please.build", "testtest"))
}

func TestSignFileBadKeyring(t *testing.T) {
	assert.Error(t, SignFile(testTxt, "test.txt.asc", badTxt, "test@please.build", "testtest"))
}

func TestSignFileMissingInput(t *testing.T) {
	assert.Error(t, SignFile("doesnt_exist", "test.txt.asc", secKey, "test@please.build", "testtest"))
}

func TestSignFileCantOutput(t *testing.T) {
	assert.Error(t, SignFile(testTxt, "dir/doesnt/exist", secKey, "test@please.build", "testtest"))
}
