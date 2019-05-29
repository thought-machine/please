// Package fuzz implements a fuzzing entry point for asp using go-fuzz.
package fuzz

import (
	"bytes"
	"sort"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/parse/rules"
)

func isWhitelisted(err error) bool {
	msg := strings.ToLower(err.Error())
	// Whitelisted error types.
	// This is pretty crude; we should probably add some structure in our errors so we
	// can tell whether it's something we "expected" or not.
	for _, whitelist := range []string{
		"attempt to output the same file",
		"cannot start with a /",
		"duplicate build target",
		"empty source path",
		"expected 'in', not",
		"has no property",
		"illegal unicode identifier",
		"int literal is too large",
		"invalid build label",
		"invalid build target name",
		"invalid type for argument",
		"is an absolute path",
		"is not a known config value",
		"is not an absolute path",
		"is not defined",
		"list index out of range",
		"missing required argument",
		"must have exactly 1",
		"no such property",
		"non-callable object",
		"requires build of",
		"str index out of range",
		"tabs are not permitted",
		"target name is empty",
		"too many arguments",
		"unexpected indent",
		"unexpected token",
		"unknown argument",
		"unknown artifact format",
		"unknown config_setting key",
		"unknown dict key",
		"unknown symbol",
		"unterminated string literal",
	} {
		if strings.Contains(msg, whitelist) {
			return true
		}
	}
	return false
}

// Fuzz implements the interface that go-fuzz requires for fuzz testing of the parser.
func Fuzz(data []byte) int {
	// This lot is copied from src/parse/init.go to load all the builtins.
	p := asp.NewParser(core.NewBuildState(1, nil, 4, core.DefaultConfiguration()))
	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		if strings.HasSuffix(filename, ".gob") {
			srcFile := strings.TrimSuffix(filename, ".gob")
			src, _ := rules.Asset(srcFile)
			p.MustLoadBuiltins("src/parse/rules/"+srcFile, src, rules.MustAsset(filename))
		}
	}

	pkg := core.NewPackage("fuzz")
	parsed, err := p.ParseReader(pkg, bytes.NewReader(data))
	if err != nil && !isWhitelisted(err) {
		panic(err)
	}
	if parsed {
		return 1
	}
	return 0
}
