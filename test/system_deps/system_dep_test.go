package google_protobuf

import "testing"

func TestTimestampProto(t *testing.T) {
	timestamp := Timestamp{}
	// These assertions are a bit pointless, essentially compiling this test is
	// sufficient to ensure things are working OK. Just want to ensure that we
	// actually use the type.
	if timestamp.Seconds != 0 || timestamp.Nanos != 0 {
		t.Errorf("Expected zeroes: %d %d", timestamp.Seconds, timestamp.Nanos)
	}
}
