package cmap

import (
	"testing"

	"github.com/cespare/xxhash/v2"
)

// input1k is some random data, base64 encoded.
const input1k = "a6HStM5Y6YNd9rt4Fdtm9yccrIZviMx5PSp0EDR+8T1RI9MYQTZ9DozWDPuM3YEBnpyLpxQZKGSP86K14b/byTYoCcXig7Y9dXggNH3Cm6GzwZVu2oda1ZMpFX+5enS37/H0jc4pfLm/6zt/1jtlwO8OrMXsZq7Sq2pgWu0EMY/p5iioHhbXShuYnOoGUEs+6EYxtJk36pHNCARJEhVwGvBGLFuGxwJBWcb/6GJwX8YciHf7ViiGQvzuIyd4wLFg+YmwmocbyWfWcFDb1URSY3O/0Kt+84ldhGDKjWlI743Yzkbvu5+1c7uV16lNIksAi7+z9Jqs4JSmMt508empk2qfNo69ouH7miW5686qiiXmz8pQnOpiQB7JadeH0IcorFTt2rAZJCymx9Xya/7JuwD2/ISXCxeFVKtojpsJxANpfBiFn0aD7dGO3U5XuveaVNKdVcrBJF0PEsfWnHfTA3Z7q4hfnylN14JgSX2l8r0Fx7m/WDamfvgDiBamMFUZmnWmGvzFNF4h0v7M1GuvX97TwgOIt9mmy8ITXPOUvohiQl75N5HxRMF5ueh5PAwBgPTgEk0G7NID3kMXedHCSUy+ox2vjCU3PxpL19LcPmbj7tTNk+xZIPDi+RW+WU0KTWzeqEZ+hAj7X/qOruWW0nESTo43CLnXVcbrdRU9yo69Ya0uNax6P5TiW1UI3ji2yyaxkKDKjroiRJpFbsifsGfGQLoINuuYWKLtE2wAl4yG6bZodn5OYgXmFtckecR8gElvAVCTDE67sz7U2yEH/VVNMRLvyKl82JTnlrKismrxtZ9F59sSSUOLwS9ugNq0wm8yIE+dTzxHdxKJqWpYDVBpbIfmYqIjAaUkL+/lzXiF4/gIgNgaKeEKzoirTd2cpFMxrooJd2zQsr4g9fcI4m5S5Pso9aydSK/mbFdNDRNEFeVxB87YKvl8+yMigT3J4xxv2aFf8idmndaTjm5mrqkVfUn6eR22Q14fVdbUsjhyLgd7t+eZYgfpb4W39XKYmKPDH0ZTj/F/dujwbWu6eKN1Q2eg0yM4vYF4xMCyJ0YgUdiSu5CllyrlAP+D+ctybxTT6IjhkJTrf3x7jUEwZ0niGqUslVR6c6StWKhe8Do6c/noLHPXi5uDgZgXJr9H3LnlFqyphnHNXTndisN/0iZaGNct5CkZUFFbiaEOntDVwCDYtyJKbWzelZnJyi6mNHpbYFxJow5+055mG/uqGtxh4IVzGEz2QqwW1mMxwmXOuNvKn7cJ4065nPUU7KOtNRe3KzBb98iPEaZro/sSN4ildVAZ7RD2ADZWVMqMVteWsMPoVCKq70rhyGIgFI5fXx+4ITrqRXVy2CtUKsnpWg"

// input20 is a shorter piece of random data, base64 encoded.
const input20 = "6fx5haW0ty6CjwrZ+GnFZyCmGyI="

var ref1k = ref(input1k)
var ref20 = ref(input20)

func BenchmarkFnv32_1k(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := Fnv32(input1k); x != ref1k {
			b.Fatalf("Does not match reference: %d (expected %d)", x, ref1k)
		}
	}
}

func BenchmarkFnv32_20(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := Fnv32(input20); x != ref20 {
			b.Fatalf("Does not match reference: %d (expected %d)", x, ref20)
		}
	}
}

func BenchmarkFnv32_Ref1k(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := ref(input1k); x != ref1k {
			b.Fatalf("Does not match reference: %d (expected %d)", x, ref1k)
		}
	}
}

func BenchmarkFnv32_Ref20(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := ref(input20); x != ref20 {
			b.Fatalf("Does not match reference: %d (expected %d)", x, ref20)
		}
	}
}

func BenchmarkFnv32_1kXor(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := Fnv32(input1k) ^ Fnv32(input1k) ^ Fnv32(input1k); x == initial { //nolint:staticcheck
			b.Fatalf("incorrect hash")
		}
	}
}

func BenchmarkFnv32_1kS(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := Fnv32s(input1k, input1k, input1k); x == initial {
			b.Fatalf("incorrect hash")
		}
	}
}

func BenchmarkFnv32_20Xor(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := Fnv32(input20) ^ Fnv32(input20) ^ Fnv32(input20); x == initial { //nolint:staticcheck
			b.Fatalf("incorrect hash")
		}
	}
}

func BenchmarkFnv32_20S(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if x := Fnv32s(input20, input20, input20); x == initial {
			b.Fatalf("incorrect hash")
		}
	}
}

func BenchmarkXXHash_1k(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		XXHash(input1k)
	}
}

func BenchmarkXXHash_20(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		XXHash(input20)
	}
}

func BenchmarkXXHash_1kS(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		XXHashes(input1k, input1k, input1k)
	}
}

func BenchmarkXXHash_20S(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		XXHashes(input20, input20, input20)
	}
}

func BenchmarkXXHash_1kSx(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		xxHashes(input1k, input1k, input1k)
	}
}

func BenchmarkXXHash_20Sx(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		xxHashes(input20, input20, input20)
	}
}

// ref is our reference implementation, from OneOfOne/cmap/hashers
// N.B. As of Go 1.18 the workaround is no longer needed (hence why we have our own), this
//
//	is reproduced verbatim as a reference implementation.
func ref(s string) (hash uint64) {
	const prime32 = 16777619
	if hash = 2166136261; s == "" {
		return
	}

	// workaround not being able to inline for loops.
	// https://github.com/golang/go/issues/21490
	i := 0
L:
	hash *= prime32
	hash ^= uint64(s[i])
	if i++; i < len(s) {
		goto L
	}

	return
}

// This is provided to compare against XXHashes
// It's approximately half the speed on our 20-char input.
func xxHashes(s ...string) uint64 {
	d := xxhash.New()
	for _, x := range s {
		d.WriteString(x)
	}
	return d.Sum64()
}

// FNV-32 code which we used to use, but no longer do.

const prime32 = 16777619
const initial = uint64(2166136261)

func Fnv32(s string) uint64 {
	hash := initial
	for i := 0; i < len(s); i++ {
		hash *= prime32
		hash ^= uint64(s[i])
	}
	return hash
}

func Fnv32s(s ...string) (hash uint64) {
	for _, x := range s {
		for i := 0; i < len(x); i++ {
			hash *= prime32
			hash ^= uint64(x[i])
		}
	}
	return hash
}
