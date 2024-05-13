package fs

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortPathsBasic(t *testing.T) {
	assert.Equal(t, []string{
		"src/fs",
		"src/fs/test_data/data.txt",
		"src/fs/hash.go",
		"src/fs/sort.go",
	}, SortPaths([]string{
		"src/fs/sort.go",
		"src/fs/hash.go",
		"src/fs",
		"src/fs/test_data/data.txt",
	}))
}

func TestSortPathsBasic2(t *testing.T) {
	assert.Equal(t, []string{
		"common/python/aws/s3.py",
		"common/python/aws/ses.py",
		"common/python/async_unblock.py",
		"common/python/boto.py",
	}, SortPaths([]string{
		"common/python/aws/ses.py",
		"common/python/aws/s3.py",
		"common/python/async_unblock.py",
		"common/python/boto.py",
	}))
}

func TestSortPaths2(t *testing.T) {
	for i := 0; i < 100; i++ {
		assert.Equal(t, []string{
			"common/js/Analytics/AppAnalytics.web.js",
			"common/protos/categories.py",
			"common/protos/categories_proto.js",
			"common/python/resources.py",
			"common/python/so_import.py",
			"enterprise_platform/proto/users_model_pb_light.js",
			"infrastructure/dependency_visualiser/JsonFormatter.js",
		}, SortPaths(shuffle([]string{
			"common/python/so_import.py",
			"common/protos/categories.py",
			"common/protos/categories_proto.js",
			"common/python/resources.py",
			"infrastructure/dependency_visualiser/JsonFormatter.js",
			"enterprise_platform/proto/users_model_pb_light.js",
			"common/js/Analytics/AppAnalytics.web.js",
		})))
	}
}

func shuffle(s []string) []string {
	for i := len(s); i > 0; i-- {
		j := rand.IntN(i)
		s[i-1], s[j] = s[j], s[i-1]
	}
	return s
}
