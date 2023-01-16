// Package main implements an OpenPGP-compatible signer for our releases.
// It's ultimately easier to have our own, given a solid upstream library for it,
// than managing cross-platform issues with builds.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sigstore/sigstore/pkg/signature/kms/gcp"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/release_signer/signer"
)

var opts = struct {
	Usage string
	PGP   struct {
		Out      string `short:"o" long:"output" env:"OUT" description:"Output filename (signature)"`
		In       string `short:"i" long:"input" description:"Input file to sign"`
		Key      string `short:"k" long:"key" env:"PLZ_GPG_KEY" description:"Private ASCII-armoured key file to sign with" required:"true"`
		User     string `short:"u" long:"user" default:"releases@please.build" description:"User to sign for"`
		Password string `short:"p" long:"password" env:"GPG_PASSWORD" required:"true" description:"Password to unlock keyring"`
		Args     struct {
			Files []string `positional-arg-name:"files" description:"A list of files to sign"`
		} `positional-args:"true"`
	} `command:"pgp" description:"Signs the binary with a pgp key"`
	KMS struct {
		Out  string `short:"o" long:"output" env:"OUT" description:"Output filename (signature)"`
		In   string `short:"i" long:"input" description:"Input file to sign"`
		Key  string `short:"k" long:"key" env:"PLZ_KMS_KEY" description:"The kms key resource name with a scheme e.g. gcpkms://" required:"true"`
		Args struct {
			Files []string `positional-arg-name:"files" description:"A list of files to sign"`
		} `positional-args:"true"`
	} `command:"kms" description:"signs the binary with a key stored in a KMS"`
}{
	Usage: `
release_signer is an internal tool used to sign Please releases with.

All it can do is create an ASCII-armoured detached signature for a single file.
`,
}

func main() {
	cmd := cli.ParseFlagsOrDie("release_signer", &opts)
	if cmd == "kms" {
		kms()
	} else {
		pgp()
	}
}

func pgp() {
	if len(opts.PGP.Args.Files) > 0 {
		for _, f := range opts.PGP.Args.Files {
			if err := signer.SignFileWithPGP(f, f+".asc", opts.PGP.Key, opts.PGP.User, opts.PGP.Password); err != nil {
				fmt.Fprintf(os.Stderr, "Signing failed: %s\n", err)
				os.Exit(1)
			}
		}
	} else {
		if opts.PGP.In == "" {
			fmt.Fprintln(os.Stderr, "You must either provide a list of files to sign or --input")
			os.Exit(1)
		}
		if err := signer.SignFileWithPGP(opts.PGP.In, opts.PGP.Out, opts.PGP.Key, opts.PGP.User, opts.PGP.Password); err != nil {
			fmt.Fprintf(os.Stderr, "Signing failed: %s\n", err)
			os.Exit(1)
		}
	}
	os.Exit(0)
}

func kms() {
	gcpSigner, err := gcp.LoadSignerVerifier(context.Background(), opts.KMS.Key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Signing failed: %s\n", err)
		os.Exit(1)
	}

	if len(opts.KMS.Args.Files) > 0 {
		for _, f := range opts.KMS.Args.Files {
			if err := signer.SignFileWithSigner(f, f+".sig", gcpSigner); err != nil {
				fmt.Fprintf(os.Stderr, "Signing failed: %s\n", err)
				os.Exit(1)
			}
		}
	} else {
		if opts.KMS.In == "" {
			fmt.Fprintln(os.Stderr, "You must either provide a list of files to sign or --input")
			os.Exit(1)
		}
		if err := signer.SignFileWithSigner(opts.KMS.In, opts.KMS.Out, gcpSigner); err != nil {
			fmt.Fprintf(os.Stderr, "Signing failed: %s\n", err)
			os.Exit(1)
		}
	}
	os.Exit(0)
}
