package covertest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pb "github.com/thought-machine/please/test/go_rules/cover-test/test-proto"
)

func TestProto(t *testing.T) {
	// Not much to be done here, the real test is that this has compiled.
	msg := &pb.Test{S: "test"}
	assert.Equal(t, "test", msg.S)
}
