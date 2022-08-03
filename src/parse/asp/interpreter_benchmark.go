package asp

import (
	"fmt"
	"sort"
	"testing"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
)

func BenchmarkParseFile(b *testing.B) {
	b.ReportAllocs()
	state := core.NewDefaultBuildState()
	parser := NewParser(state)
	dir, _ := rules.AllAssets(map[string]struct{}{})
	sort.Strings(dir)
	for _, filename := range dir {
		src, _ := rules.ReadAsset(filename)
		parser.MustLoadBuiltins(filename, src)
	}
	for i := 0; i < b.N; i++ {
		pkg := core.NewPackage(fmt.Sprintf("benchmark_%d", i))
		if err := parser.ParseFile(pkg, nil, nil, false, "src/parse/asp/test_data/benchmark_parse_file.build"); err != nil {
			panic(err)
		}
	}
}
