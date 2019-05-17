// +build !bootstrap

package update

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/thought-machine/please/src/cli"
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
	return bufio.NewReader(cli.NewProgressReader(ioutil.NopCloser(bytes.NewReader(b)), len(b), "Extracting"))
}

// mustVerifyHash verifies the sha256 hash of the downloaded file matches one of the given ones.
// On success it returns an equivalent reader, on failure it panics.
func mustVerifyHash(r io.Reader, hashes []string) io.Reader {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	log.Notice("Verifying hash of downloaded tarball")
	sum := sha256.Sum256(b)
	checksum := hex.EncodeToString(sum[:])
	for _, hash := range hashes {
		if hash == checksum {
			log.Notice("Good checksum: %s", checksum)
			return bytes.NewReader(b)
		}
	}
	if len(hashes) == 1 {
		panic(fmt.Errorf("Invalid checksum of downloaded file, was %s, expected %s", checksum, hashes[0]))
	}
	panic(fmt.Errorf("Invalid checksum of downloaded file, was %s, expected one of [%s]", checksum, strings.Join(hashes, ", ")))
}
