package core

import "github.com/thought-machine/go-flags"

// AttachAliasFlags is disabled during initial bootstrap.
func (config *Configuration) AttachAliasFlags(parser *flags.Parser) bool {
	return false
}
