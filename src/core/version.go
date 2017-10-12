package core

import "github.com/coreos/go-semver/semver"

// PleaseVersion is the current version of Please.
// Note that non-bootstrap builds replace this interim version with a real one.
var PleaseVersion = *semver.New("1.0.9999")
