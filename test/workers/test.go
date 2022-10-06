package main

import (
	"bytes"
	"io"
	"net/http"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("worker_test")

func main() {
	resp, err := http.Get("http://127.0.0.1:31812")
	if err != nil {
		log.Fatalf("Failed to get: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unsuccessful get: %s", resp.Status)
	}
	defer resp.Body.Close()
	if b, err := io.ReadAll(resp.Body); err != nil {
		log.Fatalf("Failed to read: %s", err)
	} else if !bytes.Equal([]byte("kitten!"), b) {
		log.Fatalf("Unexpected response: %s", b)
	}
}
