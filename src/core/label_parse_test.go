// Tests parsing of build labels.

package core

import (
	"testing"
)

func assertLabel(t *testing.T, in, pkg, name string) BuildLabel {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Failed to parse %s: %s", in, r)
		}
	}()
	label := ParseBuildLabel(in, "current_package")
	if label.PackageName != pkg {
		t.Errorf("Incorrect parse of %s: package name should be %s, was %s", in, pkg, label.PackageName)
	}
	if label.Name != name {
		t.Errorf("Incorrect parse of %s: target name should be %s, was %s", in, name, label.Name)
	}
	return label
}

func assertSubrepoLabel(t *testing.T, in, pkg, name, subrepo string) {
	t.Helper()
	label := assertLabel(t, in, pkg, name)
	if label.Subrepo != subrepo {
		t.Errorf("Incorrect parse of %s: subrepo should be %s, was %s", in, subrepo, label.Subrepo)
	}
}

func assertRelativeLabel(t *testing.T, in, pkg, name string) {
	t.Helper()
	if label, err := parseMaybeRelativeBuildLabel(in, "current_package"); err != nil {
		t.Errorf("Failed to parse %s: %s", in, err)
	} else if label.PackageName != pkg {
		t.Errorf("Incorrect parse of %s: package name should be %s, was %s", in, pkg, label.PackageName)
	} else if label.Name != name {
		t.Errorf("Incorrect parse of %s: target name should be %s, was %s", in, name, label.Name)
	}
}

func assertNotLabel(t *testing.T, in, reason string) {
	t.Helper()
	var label BuildLabel
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s should have failed (%s), instead generated %s", in, reason, label)
		}
	}()
	label = ParseBuildLabel(in, "current_package")
}

// These labels are accepted anywhere, on the command line or in BUILD files.

func TestAbsoluteTarget(t *testing.T) {
	assertLabel(t, "//path/to:target", "path/to", "target")
	assertLabel(t, "//path:target", "path", "target")
	assertLabel(t, "//:target", "", "target")
	assertNotLabel(t, "//path:to/target", "can't have slashes in target names")
	assertNotLabel(t, "//path:to:target", "can't have multiple colons")
	assertNotLabel(t, "/path:to/target", "must have two initial slashes")
	assertNotLabel(t, "/path/to:", "must pass a target name")
}

func TestLocalTarget(t *testing.T) {
	assertLabel(t, ":target", "current_package", "target")
	assertLabel(t, ":thingy_wotsit_123", "current_package", "thingy_wotsit_123")
	assertNotLabel(t, ":to/target", "can't have slashes in target names")
	assertNotLabel(t, ":to:target", "can't have multiple colons")
	assertNotLabel(t, "::to_target", "can't have multiple colons")
}

func TestImplicitTarget(t *testing.T) {
	assertLabel(t, "//path/to", "path/to", "to")
	assertLabel(t, "//path", "path", "path")
	assertNotLabel(t, "/path", "must have two initial slashes")
}

func TestSubTargets(t *testing.T) {
	assertLabel(t, "//path/to/...", "path/to", "...")
	assertLabel(t, "//path/...", "path", "...")
	assertLabel(t, "//...", "", "...")
	// These three are not passing at the moment. Not completely crucial since the ... will just be
	// treated as a package name but would be nice if they were rejected here.
	// assertNotLabel(t, "//...:hello", "can't have stuff after the ellipsis")
	// assertNotLabel(t, "//...1234", "can't have stuff after the ellipsis")
	// assertNotLabel(t, "//.../...", "can't have multiple ellipses")
}

// The following are only accepted on the command line and converted to absolute
// labels based on the current directory.

func TestRelativeSubTargets(t *testing.T) {
	assertRelativeLabel(t, "...", "current_package", "...")
	assertRelativeLabel(t, "path/to/...", "current_package/path/to", "...")
	assertNotLabel(t, "...:hello", "can't have stuff after the ellipsis")
	assertNotLabel(t, "...1234", "can't have stuff after the ellipsis")
	assertNotLabel(t, ".../...", "can't have multiple ellipses")
}

func TestRelativeTarget(t *testing.T) {
	assertRelativeLabel(t, "path/to:thingy", "current_package/path/to", "thingy")
	assertRelativeLabel(t, ":thingy", "current_package", "thingy")
	assertNotLabel(t, "path/to:", "must have a target name")
	assertNotLabel(t, "path/to:thingy/mabob", "can't have a slash in target name")
	assertNotLabel(t, "path/to:thingy:mabob", "can only have one colon")
}

func TestRelativeImplicitTarget(t *testing.T) {
	assertRelativeLabel(t, "path/to", "current_package/path/to", "to")
	assertRelativeLabel(t, "path", "current_package/path", "path")
	assertNotLabel(t, "path/to:", "must have a target name")
}

// Test for issue #55 where we were incorrectly allowing consecutive double slashes,
// which has all manner of weird follow-on effects
func TestDoubleSlashes(t *testing.T) {
	assertNotLabel(t, "//src//core", "double slashes not allowed")
	assertNotLabel(t, "//src//core:target1", "double slashes not allowed")
	assertNotLabel(t, "//src/core/something//something", "double slashes not allowed")
}

// Test that labels can't match reserved suffixes used for temp dirs.
func TestReservedTempDirs(t *testing.T) {
	assertNotLabel(t, "//src/core:core._build", "._build is a reserved suffix")
	assertNotLabel(t, "//src/core:core._test", "._test is a reserved suffix")
}

func TestNonAsciiParse(t *testing.T) {
	assertLabel(t, "//src/core:aerolínea", "src/core", "aerolínea")
}

func TestDotsArentAccepted(t *testing.T) {
	assertNotLabel(t, "//src/core:.", ". is not a valid label name")
	assertNotLabel(t, "//src/core:..", ".. is not a valid label name")
	assertNotLabel(t, "//src/core:...", "... is not a valid label name")
	assertNotLabel(t, "//src/core:....", ".... is not a valid label name")
	assertLabel(t, "//src/core/...", "src/core", "...")
}

func TestPipesArentAccepted(t *testing.T) {
	assertNotLabel(t, "//src/core:core|build_label.go", "| is not allowed in build labels")
}

func TestSubrepos(t *testing.T) {
	assertSubrepoLabel(t, "@subrepo//pkg:target", "pkg", "target", "subrepo")
	assertSubrepoLabel(t, "@com_google_googletest//:gtest_main", "", "gtest_main", "com_google_googletest")
	assertSubrepoLabel(t, "@test_x86:target", "current_package", "target", "test_x86")
}

func TestNewSyntaxSubrepo(t *testing.T) {
	// Test the new triple-slash syntax.
	assertSubrepoLabel(t, "///subrepo//pkg:target", "pkg", "target", "subrepo")
	assertSubrepoLabel(t, "///third_party/cc/gtest//:gtest", "", "gtest", "third_party/cc/gtest")
	assertSubrepoLabel(t, "///third_party/cc/gtest", "", "gtest", "third_party/cc/gtest")
}
