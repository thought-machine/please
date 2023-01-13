package signer

import (
	"crypto"
	"os"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/stretchr/testify/assert"
)

const (
	pubKey  = "tools/release_signer/signer/test_data/pub.gpg"
	secKey  = "tools/release_signer/signer/test_data/sec.gpg"
	pemKey  = "tools/release_signer/signer/test_data/key.pem"
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
	_, err = openpgp.CheckArmoredDetachedSignature(entities, f1, f2, nil)
	return err == nil
}

func verifyWithVerifier(message, signature string, verifier signature.Verifier) bool {
	msgFile, err := os.Open(message)
	must(err)
	signatureFile, err := os.Open(signature)
	must(err)

	return verifier.VerifySignature(signatureFile, msgFile) == nil
}

func signerVerifier() signature.SignerVerifier {
	signer, err := signature.LoadSignerVerifierFromPEMFile(pemKey, crypto.SHA256, cryptoutils.StaticPasswordFunc([]byte("testtest")))
	must(err)
	return signer
}

func TestSignFile(t *testing.T) {
	t.Run("pgp", func(t *testing.T) {
		assert.NoError(t, SignFileWithPGP(testTxt, "test.txt.asc", secKey, "test@please.build", "testtest"))
		assert.True(t, verifyFile(testTxt, "test.txt.asc", pubKey))
	})
	t.Run("sigstore", func(t *testing.T) {
		s := signerVerifier()
		assert.NoError(t, SignFileWithSigner(testTxt, "test.txt.sig", s))
		assert.True(t, verifyWithVerifier(testTxt, "test.txt.sig", s))
	})
}

func TestSignFileBadPassphrase(t *testing.T) {
	assert.Error(t, SignFileWithPGP(testTxt, "test.txt.asc", secKey, "test@please.build", "nope"))
}

func TestSignFileBadSignature(t *testing.T) {
	t.Run("pgp", func(t *testing.T) {
		assert.NoError(t, SignFileWithPGP(testTxt, "test.txt.asc", secKey, "test@please.build", "testtest"))
		assert.False(t, verifyFile(badTxt, "test.txt.asc", pubKey))
	})
	t.Run("sigstore", func(t *testing.T) {
		s := signerVerifier()
		assert.NoError(t, SignFileWithSigner(testTxt, "test.txt.sig", s))
		assert.False(t, verifyWithVerifier(badTxt, "test.txt.asc", s))
	})
}

func TestSignFileUnknownUser(t *testing.T) {
	assert.Error(t, SignFileWithPGP(testTxt, "test.txt.asc", secKey, "not@please.build", "testtest"))
}

func TestSignFileMissingKeyring(t *testing.T) {
	assert.Error(t, SignFileWithPGP(testTxt, "test.txt.asc", "doesnt_exist", "test@please.build", "testtest"))
}

func TestSignFileBadKeyring(t *testing.T) {
	assert.Error(t, SignFileWithPGP(testTxt, "test.txt.asc", badTxt, "test@please.build", "testtest"))
}

func TestSignFileMissingInput(t *testing.T) {
	t.Run("pgp", func(t *testing.T) {
		assert.Error(t, SignFileWithPGP("doesnt_exist", "test.txt.asc", secKey, "test@please.build", "testtest"))
	})
	t.Run("sigstore", func(t *testing.T) {
		s := signerVerifier()
		assert.Error(t, SignFileWithSigner("doesnt_exist", "test.txt.asc", s))
	})
}

func TestSignFileCantOutput(t *testing.T) {
	t.Run("pgp", func(t *testing.T) {
		assert.Error(t, SignFileWithPGP(testTxt, "dir/doesnt/exist", secKey, "test@please.build", "testtest"))
	})
	t.Run("sigstore", func(t *testing.T) {
		s := signerVerifier()
		assert.Error(t, SignFileWithSigner(testTxt, "dir/doesnt/exist", s))
	})
}
