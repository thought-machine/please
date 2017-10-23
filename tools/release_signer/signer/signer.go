package signer

import (
	"fmt"
	"os"

	"golang.org/x/crypto/openpgp"
)

// SignFile creates a detached ASCII-armoured signature for the given file.
func SignFile(filename, output, keyring, user, password string) error {
	f, err := os.Open(keyring)
	if err != nil {
		return err
	}
	defer f.Close()
	entities, err := openpgp.ReadArmoredKeyRing(f)
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
