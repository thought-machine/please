// +build bootstrap

package core

import "github.com/jessevdk/go-flags"

// AttachAliasFlags is disabled during initial bootstrap.
func (config *Configuration) AttachAliasFlags(parser *flags.Parser) bool {
	return false
}
