package maven

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func v(s string) *Version {
	ver := &Version{}
	ver.Unmarshal(s)
	return ver
}

func TestUnmarshal(t *testing.T) {
	version := v("7")
	assert.Equal(t, 7, version.Min.Major)
	assert.Equal(t, 0, version.Min.Minor)
	assert.Equal(t, 0, version.Min.Incremental)

	version = v("1.2.3")
	assert.Equal(t, 1, version.Min.Major)
	assert.Equal(t, 2, version.Min.Minor)
	assert.Equal(t, 3, version.Min.Incremental)
	assert.True(t, version.Min.Inclusive)
	assert.Equal(t, 0, version.Max.Minor)
	assert.Equal(t, 0, version.Max.Incremental)

	version = v("[1.2.3]")
	assert.Equal(t, 1, version.Min.Major)
	assert.Equal(t, 2, version.Min.Minor)
	assert.Equal(t, 3, version.Min.Incremental)
	assert.True(t, version.Min.Inclusive)
	assert.Equal(t, 1, version.Max.Major)
	assert.Equal(t, 2, version.Max.Minor)
	assert.Equal(t, 3, version.Max.Incremental)
	assert.True(t, version.Max.Inclusive)

	version = v("[1.2.3,2.3.4)")
	assert.Equal(t, 1, version.Min.Major)
	assert.Equal(t, 2, version.Min.Minor)
	assert.Equal(t, 3, version.Min.Incremental)
	assert.True(t, version.Min.Inclusive)
	assert.Equal(t, 2, version.Max.Major)
	assert.Equal(t, 3, version.Max.Minor)
	assert.Equal(t, 4, version.Max.Incremental)
	assert.False(t, version.Max.Inclusive)
}

func TestVersionsLessThan(t *testing.T) {
	assert.True(t, v("0.9").Matches(v("(,1.0]")))
	assert.True(t, v("1.0").Matches(v("(,1.0]")))
	assert.False(t, v("1.1").Matches(v("(,1.0]")))
}

func TestVersionsGreaterThan(t *testing.T) {
	assert.True(t, v("1.5").Matches(v("[1.5,)")))
	assert.True(t, v("1.6").Matches(v("[1.5,)")))
	assert.False(t, v("1.4").Matches(v("[1.5,)")))
}

func TestVersionsImplicit(t *testing.T) {
	assert.False(t, v("0.9").Matches(v("1.0")))
	assert.True(t, v("1.0").Matches(v("1.0")))
	assert.True(t, v("1.1").Matches(v("1.0")))
}

func TestVersionsExact(t *testing.T) {
	assert.False(t, v("0.9").Matches(v("[1.0]")))
	assert.True(t, v("1.0").Matches(v("[1.0]")))
	assert.False(t, v("1.1").Matches(v("[1.0]")))
	assert.False(t, v("1.0-SNAPSHOT").Matches(v("[1.0]")))
}

func TestVersionsRangeInclusive(t *testing.T) {
	assert.False(t, v("1.0").Matches(v("[1.2,1.3]")))
	assert.True(t, v("1.2").Matches(v("[1.2,1.3]")))
	assert.True(t, v("1.2.5").Matches(v("[1.2,1.3]")))
	assert.True(t, v("1.3").Matches(v("[1.2,1.3]")))
	assert.False(t, v("1.4").Matches(v("[1.2,1.3]")))
}

func TestVersionsRangeExclusive(t *testing.T) {
	assert.False(t, v("0.9").Matches(v("[1.0,2.0)")))
	assert.True(t, v("1.0").Matches(v("[1.0,2.0)")))
	assert.True(t, v("1.5").Matches(v("[1.0,2.0)")))
	assert.True(t, v("1.9.99").Matches(v("[1.0,2.0)")))
	assert.False(t, v("2.0").Matches(v("[1.0,2.0)")))
}

func TestIntersect(t *testing.T) {
	ver := v("[1.0,3.0]")
	assert.True(t, ver.Intersect(v("[2.0,3.0]")))
	assert.Equal(t, 2, ver.Min.Major)
	assert.Equal(t, 0, ver.Min.Minor)
	assert.Equal(t, 3, ver.Max.Major)
	assert.Equal(t, 0, ver.Max.Minor)
	assert.True(t, v("[2.0]").Matches(ver))
	assert.False(t, v("[1.0]").Matches(ver))
	assert.True(t, v("[2.5.4]").Matches(ver))
	assert.True(t, v("[3.0]").Matches(ver))
	assert.False(t, v("[3.1]").Matches(ver))
}

func TestIntersectUnparseable(t *testing.T) {
	ver := v("1.0.1B")
	assert.True(t, ver.Intersect(v("1.1")))
	assert.Equal(t, 1, ver.Min.Major)
	assert.Equal(t, 1, ver.Min.Minor)
	assert.Equal(t, 0, ver.Min.Incremental)
	assert.Equal(t, "", ver.Min.Qualifier)
}
