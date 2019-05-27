package main

import (
	"encoding/json"
	"net/http"
	"os"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("worker")

// A BuildMessage is a minimal subset of BuildRequest / BuildResponse that we use here.
type BuildMessage struct {
	Rule    string `json:"rule"`
	Success bool   `json:"success"`
}

func main() {
	// Start a web server that we use to communicate with the other tests.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("kitten!"))
	})
	go http.ListenAndServe(":31812", nil)

	// Now loop, reading requests forever.
	// These are just used to indicate that we're ready to receive a new test.
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		msg := &BuildMessage{}
		if err := decoder.Decode(msg); err != nil {
			log.Fatalf("Failed to decode input: %s", err)
		}
		msg.Success = true
		if err := encoder.Encode(msg); err != nil {
			log.Fatalf("Failed to encode output: %s", err)
		}
	}
}
