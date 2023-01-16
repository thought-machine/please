package signer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/sigstore/sigstore/pkg/signature"
)

// SignFileWithPGP creates a detached signature. This method is insecure has the private key was potentially compromised
// in the CircleCI leak. Going forward, all verification is based off the signature returned from SignFileWithSigner.
// This signature is only included to allow older Please versions to update.
func SignFileWithPGP(filename, output, keyring, user, password string) error {
	if strings.HasPrefix(keyring, "-----BEGIN PGP") {
		// Keyring is an actual key, not a file.
		return signFile(filename, output, user, password, strings.NewReader(keyring))
	} else if strings.HasPrefix(keyring, "LS0tLS1") {
		// Keyring is a base64 encoded key
		b, err := base64.StdEncoding.DecodeString(keyring)
		if err != nil {
			return err
		}
		return signFile(filename, output, user, password, bytes.NewReader(b))
	}
	f, err := os.Open(keyring)
	if err != nil {
		return err
	}
	defer f.Close()
	return signFile(filename, output, user, password, f)
}

// SignFileWithSigner signs a file with the provided signer. This is populated by a kms backed signer in main.go so we
// never actually handle the private key.
func SignFileWithSigner(filename, output string, signer signature.Signer) error {
	message, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	sig, err := signer.SignMessage(bytes.NewReader(message))
	if err != nil {
		return err
	}

	return os.WriteFile(output, sig, 0444)
}

func signFile(filename, output, user, password string, keyring io.Reader) error {
	entities, err := openpgp.ReadArmoredKeyRing(keyring)
	if err != nil {
		return err
	}
	signer, err := findSigningEntity(entities, user)
	if err != nil {
		return err
	}
	if err := signer.PrivateKey.Decrypt([]byte(password)); err != nil {
		return err
	}
	w, err := os.Create(output)
	if err != nil {
		return err
	}
	defer w.Close()
	f2, err := os.Open(filename)
	if err != nil {
		return err
	}
	return openpgp.ArmoredDetachSign(w, signer, f2, nil)
}

// findSigningEntity finds the entity in a list with the given name.
func findSigningEntity(entities openpgp.EntityList, user string) (*openpgp.Entity, error) {
	for _, entity := range entities {
		for name, identity := range entity.Identities {
			if name == user || identity.UserId.Name == user || identity.UserId.Email == user {
				return entity, nil
			}
		}
	}
	return nil, fmt.Errorf("No entity found for the given user")
}
