// Small stress test for the cache to help flush out concurrency bugs.
package server

import (
	"crypto/sha1"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/op/go-logging.v1"
)

var cache *Cache

const size = 1000

func init() {
	logging.SetLevel(logging.NOTICE, "server")
	cache = newCache("cache")
}

func TestStoreFiles(t *testing.T) {
	// Store 1000 files in parallel
	var wg sync.WaitGroup
	wg.Add(size)
	for _, i := range rand.Perm(size) {
		go func(i int) {
			path, contents := artifact(i)
			assert.NoError(t, cache.StoreArtifact(path, contents, ""))
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestRetrieveFiles(t *testing.T) {
	// Store 1000 files in parallel
	var wg sync.WaitGroup
	wg.Add(size)
	for _, i := range rand.Perm(size) {
		go func(i int) {
			path, contents := artifact(i)
			arts, err := cache.RetrieveArtifact(path)
			if os.IsNotExist(err) { // It's allowed not to exist.
				wg.Done()
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, 1, len(arts))
			assert.Equal(t, contents, arts[0].Body)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestDeleteFiles(t *testing.T) {
	// Delete 1000 files in parallel
	var wg sync.WaitGroup
	wg.Add(size)
	for _, i := range rand.Perm(size) {
		go func(i int) {
			path, _ := artifact(i)
			assert.NoError(t, cache.DeleteArtifact(path))
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestInParallel(t *testing.T) {
	// Run all above tests in parallel.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		TestStoreFiles(t)
		wg.Done()
	}()
	go func() {
		TestRetrieveFiles(t)
		wg.Done()
	}()
	go func() {
		TestDeleteFiles(t)
		wg.Done()
	}()
	wg.Wait()
}

func TestInParallelWithCleaning(t *testing.T) {
	// Run store & retrieve tests in parallel & cleaning at the same time.
	// It's too awkward to write a reliable test with deleting too and actually
	// guarantee that the cleaner does anything.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		TestStoreFiles(t)
		wg.Done()
	}()
	go func() {
		TestRetrieveFiles(t)
		wg.Done()
	}()
	go func() {
		// Each artifact is sha1.Size bytes long, so this should guarantee it runs once.
		for !cache.singleClean(sha1.Size*size/10, sha1.Size*size/2) {
		}
		// Now run it a bunch more times.
		for i := 0; i < 100; i++ {
			cache.singleClean(sha1.Size*size/10, sha1.Size*size/2)
		}
		wg.Done()
	}()
	wg.Wait()
}

func artifact(i int) (string, []byte) {
	path := fmt.Sprintf("src/%d/%d/%d.dat", i/100, i/10, i)
	contents := sha1.Sum([]byte(strconv.Itoa(i)))
	return path, contents[:]
}
