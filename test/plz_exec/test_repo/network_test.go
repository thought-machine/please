package main

import (
	"log"
	"net"
	"time"
)

func main() {
	if _, err := net.DialTimeout("tcp", "google.com:80", 1*time.Second); err != nil {
		log.Fatal(err)
	}
}
