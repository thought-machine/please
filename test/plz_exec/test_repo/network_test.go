package main

import (
	"log"
	"net"
	"time"
)

func main() {
	if _, err := net.DialTimeout("tcp", "google.com:80", time.Second); err != nil {
		log.Fatalf("Dial error: %s", err)
	}
}
