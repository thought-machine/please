package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	if _, err := net.DialTimeout("tcp", "google.com:80", time.Second); err != nil {
		log.Fatalf("Network error: %s", err)
	}
	fmt.Printf("Debugging what's happening in macos")
}
