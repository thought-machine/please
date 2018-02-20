// +build !bootstrap

package update

import (
	"bytes"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/openpgp"
)

// identity is the signing identity of this key.
const identity = "Please Releases <releases@please.build>"

// verifySignature verifies an OpenPGP detached signature of a file.
// It returns true if the signature is correct according to our key.
func verifySignature(signed, signature io.Reader) bool {
	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(MustAsset("pubkey.gpg.asc")))
	if err != nil {
		log.Fatalf("%s", err) // Shouldn't happen
	}
	signer, err := openpgp.CheckArmoredDetachedSignature(entities, signed, signature)
	if err != nil {
		log.Error("Bad signature: %s", err)
		return false
	}
	log.Notice("Good signature from %s", signer.Identities[identity].UserId.Email)
	return true
}

// verifyDownload fetches a detached signature for a download and verifies it's OK.
// It returns a reader to the verified content.
func verifyDownload(signed io.Reader, url string) io.Reader {
	signature := mustDownload(url+".asc", false)
	defer signature.Close()
	return mustVerifySignature(signed, signature)
}

// mustVerifySignature verifies an OpenPGP detached signature of a file.
// It panics if the signature is not correct.
// On success it returns an equivalent reader to the original.
func mustVerifySignature(signed, signature io.Reader) io.Reader {
	// We need to be able to reuse the body again afterwards so we have to
	// download the original into a buffer.
	b, err := ioutil.ReadAll(signed)
	if err != nil {
		panic(err)
	}
	log.Notice("Verifying signature of downloaded tarball...")
	if !verifySignature(bytes.NewReader(b), signature) {
		panic("Invalid signature on downloaded file, possible tampering; will not continue.")
	}
	return bytes.NewReader(b)
}
