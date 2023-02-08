package main

import (
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("worker_test")

func main() {
	log.Errorf("Expected to fail by now")
}
