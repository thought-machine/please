package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkHashTree(b *testing.B) {
	const expected = "0dfc40ae0c5b728b2dd0797b99cf8e7361796fead051bc8f8dcfe21b2a307903"
	data := os.Getenv("DATA")

	// Calculate size of dir for metrics later
	size := 0
	if err := filepath.Walk(data, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += int(info.Size())
		}
		return err
	}); err != nil {
		b.Fatalf("Failed to calculate size of input tree: %s", err)
	}
	// Run one hash initially to ensure any fs caching is warm.
	NewPathHasher(".", false, sha256.New, "sha256").Hash(data, false, false)
	b.ResetTimer()

	for _, parallelism := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("%dWay", parallelism), func(b *testing.B) {
			start := time.Now()
			for i := 0; i < b.N; i++ {
				// N.B. We force off xattrs to avoid it trying to short-circuit anything.
				hasher := NewPathHasher(".", false, sha256.New, "sha256")
				if hash, err := hasher.Hash(data, false, false); err != nil {
					b.Fatalf("Failed to hash path %s: %s", data, err)
				} else if enc := hex.EncodeToString(hash); enc != expected {
					b.Fatalf("Unexpected hash; was %s, expected %s", enc, expected)
				}
			}
			b.ReportMetric(float64(size*b.N)/(1024.0 * 1024.0 * time.Since(start).Seconds()), "MB/s")
		})
	}
}
