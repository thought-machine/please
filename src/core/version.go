package core

import "github.com/coreos/go-semver/semver"

// RawVersion is the unparsed raw version of Please.
const RawVersion = "1.0.9999"

// PleaseVersion is the current version of Please.
// Note that non-bootstrap builds replace this interim version with a real one.
var PleaseVersion = *semver.New(RawVersion)
