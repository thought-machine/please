package main

import (
	"encoding/binary"
	"net/http"
	"os"

	"github.com/golang/protobuf/proto"
	"gopkg.in/op/go-logging.v1"

	pb "build/proto/worker"
)

var log = logging.MustGetLogger("worker")

func main() {
	// Start a web server that we use to communicate with the other tests.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("kitten!"))
	})
	go http.ListenAndServe(":31812", nil)

	// Now loop, reading requests forever.
	// These are just used to indicate that we're ready to receive a new test.
	for {
		var size int32
		if err := binary.Read(os.Stdin, binary.LittleEndian, &size); err != nil {
			log.Fatalf("Failed to read stdin: %s", err)
		}
		buf := make([]byte, size)
		if _, err := os.Stdin.Read(buf); err != nil {
			log.Fatalf("Failed to read stdin: %s", err)
		}
		request := pb.BuildRequest{}
		if err := proto.Unmarshal(buf, &request); err != nil {
			log.Fatalf("Error unmarshaling response: %s", err)
		}
		response := pb.BuildResponse{Success: true, Rule: request.Rule}
		b, err := proto.Marshal(&response)
		if err != nil { // This shouldn't really happen
			log.Fatalf("Error serialising proto: %s", err)
		}
		binary.Write(os.Stdout, binary.LittleEndian, int32(len(b)))
		os.Stdout.Write(b)
	}
}
