package unzip

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var files = []string{
	"third_party/python/xmlrunner/__init__.py",
	"third_party/python/xmlrunner/__main__.py",
	"third_party/python/xmlrunner/builder.py",
	"third_party/python/xmlrunner/extra/",
	"third_party/python/xmlrunner/extra/__init__.py",
	"third_party/python/xmlrunner/extra/djangotestrunner.py",
	"third_party/python/xmlrunner/result.py",
	"third_party/python/xmlrunner/runner.py",
	"third_party/python/xmlrunner/unittest.py",
	"third_party/python/xmlrunner/version.py",
}

func TestExtract(t *testing.T) {
	assert.NoError(t, Extract("tools/jarcat/unzip/test_data/xmlrunner.whl", ".", "", ""))
	for _, file := range files {
		_, err := os.Stat(file)
		assert.NoError(t, err)
	}
}

func TestPrefix(t *testing.T) {
	prefix := "third_party/python"
	assert.NoError(t, Extract("tools/jarcat/unzip/test_data/xmlrunner.whl", ".", "", prefix))
	for _, file := range files {
		_, err := os.Stat(file[len(prefix)+1:])
		assert.NoError(t, err)
	}
}

func TestSpecificFile(t *testing.T) {
	assert.NoError(t, Extract("tools/jarcat/unzip/test_data/xmlrunner.whl", "wibble.py", "third_party/python/xmlrunner/result.py", ""))
	_, err := os.Stat("wibble.py")
	assert.NoError(t, err)
}
