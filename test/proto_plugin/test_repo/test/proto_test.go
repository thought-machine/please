package test

import (
	"testing"

	"github.com/though-machine/please/test/proto_plugin/test/proto"
)

func TestServiceImportable(t *testing.T) {
	_ = proto.HelloRequest{}
}
