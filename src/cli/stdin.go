package cli

import (
	"bufio"
	"os"
	"strings"
)

var seenStdin = false // Used to track that we don't try to read stdin twice
// ReadStdin reads a sequence of space-delimited words from standard input.
// Words are pushed onto the returned channel asynchronously.
func ReadStdin() <-chan string {
	c := make(chan string)
	if seenStdin {
		log.Fatalf("Repeated - on command line; can't reread stdin.")
	}
	seenStdin = true
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if s != "" {
				c <- s
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdin: %s", err)
		}
		close(c)
	}()
	return c
}

// ReadAllStdin reads standard input in its entirety to a slice.
// Since this reads it completely before returning it won't handle a slow input
// very nicely. ReadStdin is therefore preferable when possible.
func ReadAllStdin() []string {
	var ret []string
	for s := range ReadStdin() {
		ret = append(ret, s)
	}
	return ret
}
