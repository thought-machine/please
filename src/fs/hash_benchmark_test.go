package fs

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/blake3"
)

func BenchmarkHashes(b *testing.B) {
	// Data sizes in kb
	for _, size := range []int{32 * 1024, 256 * 1024, 1024 * 1024} {
		testFile := fmt.Sprintf("test%d.dat", size)
		writeTestFile(b, testFile, size)
		for name, hash := range map[string]func() hash.Hash{
			"sha1":   sha1.New,
			"sha256": sha256.New,
			"crc32":  func() hash.Hash { return hash.Hash(crc32.NewIEEE()) },
			"crc64":  func() hash.Hash { return hash.Hash(crc64.New(crc64.MakeTable(crc64.ISO))) },
			"blake3": func() hash.Hash { return blake3.New() },
			"xxhash": func() hash.Hash { return xxhash.New() },
		} {
			b.Run(fmt.Sprintf("%s/%dkb", name, size), func(b *testing.B) {
				hasher := NewPathHasher("", false, hash, name)
				start := time.Now()
				for i := 0; i < b.N; i++ {
					_, err := hasher.hash(testFile, false, false, false)
					assert.NoError(b, err)
				}
				b.ReportMetric(float64(size*b.N)/1024.0*time.Since(start).Seconds(), "MB/s")
			})
		}
	}
}

func writeTestFile(b *testing.B, filename string, sizeKB int) {
	f, err := os.Create(filename)
	require.NoError(b, err)
	defer f.Close()
	data := make([]byte, 1024)
	for i := 0; i < sizeKB; i++ {
		rand.Read(data)
		_, err := f.Write(data)
		require.NoError(b, err)
	}
}
