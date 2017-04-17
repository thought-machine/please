package system_deps_go

import (
	"testing"

	google_protobuf "test/system_deps/source_context"
)

func TestSourceContextProto(t *testing.T) {
	sc := google_protobuf.SourceContext{}
	// These assertions are a bit pointless, essentially compiling this test is
	// sufficient to ensure things are working OK. Just want to ensure that we
	// actually use the type.
	if sc.FileName != "" {
		t.Errorf("Expected empty string: %s", sc.FileName)
	}
}
