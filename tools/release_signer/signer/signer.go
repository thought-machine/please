package signer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// SignFile creates a detached ASCII-armoured signature for the given file.
func SignFile(filename, output, keyring, user, password string) error {
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
