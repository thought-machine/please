package ar

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/blakesmith/ar"
	"github.com/stretchr/testify/assert"
)

func TestCreateAr(t *testing.T) {
	err := Create([]string{"tools/jarcat/ar/test_data/test.o"}, "test.a", false, false)
	assert.NoError(t, err)

	// Now read it and the reference one back and compare them.
	f1, err := os.Open("test.a")
	assert.NoError(t, err)
	defer f1.Close()
	f2, err := os.Open("tools/jarcat/ar/test_data/test.a")
	assert.NoError(t, err)
	defer f2.Close()
	r1 := ar.NewReader(f1)
	r2 := ar.NewReader(f2)

	for {
		hdr1, err1 := r1.Next()
		hdr2, err2 := r2.Next()
		if err1 == io.EOF && err2 == io.EOF {
			break
		} else if err1 == io.EOF {
			t.Errorf("Additional file in reference that's not in ours: %s", hdr2.Name)
		} else if err2 == io.EOF {
			t.Errorf("Additional file ours that's not in the reference: %s", hdr1.Name)
		}
		assert.NoError(t, err1)
		assert.NoError(t, err2)

		content1, err := ioutil.ReadAll(r1)
		assert.NoError(t, err)
		content2, err := ioutil.ReadAll(r2)
		assert.NoError(t, err)

		assert.Equal(t, content1, content2)
	}
}
