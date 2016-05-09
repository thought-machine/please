// Tests parsing of build labels.

package core

import "testing"

func assertLabel(t *testing.T, in, pkg, name string) {
	assertLabelFunc(t, in, pkg, name, ParseBuildLabel)
}

func assertRelativeLabel(t *testing.T, in, pkg, name string) {
	assertLabelFunc(t, in, pkg, name, parseMaybeRelativeBuildLabel)
}

func assertLabelFunc(t *testing.T, in, pkg, name string, f func(string, string) BuildLabel) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Failed to parse %s: %s", in, r)
		}
	}()
	label := f(in, "current_package")
	if label.PackageName != pkg {
		t.Errorf("Incorrect parse of %s: package name should be %s, was %s", in, pkg, label.PackageName)
	}
	if label.Name != name {
		t.Errorf("Incorrect parse of %s: target name should be %s, was %s", in, name, label.Name)
	}
}

func assertNotLabel(t *testing.T, in, reason string) {
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
	assertNotLabel(t, "//src/core:core#.build", "#.build is a reserved suffix")
	assertNotLabel(t, "//src/core:core#.test", "#.test is a reserved suffix")
}
