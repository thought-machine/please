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

// StdinStrings is a type used for flags; it accepts a slice of strings but also
// if it's a single - it reads its contents from stdin.
type StdinStrings []string

// Get reads stdin if needed and returns the contents of this slice.
func (s StdinStrings) Get() []string {
	if len(s) == 1 && s[0] == "-" {
		return ReadAllStdin()
	} else if ContainsString("-", s) {
		log.Fatalf("Cannot pass - to read stdin along with other arguments.")
	}
	return s
}

// ContainsString returns true if the given slice contains an individual string.
func ContainsString(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}
	return false
}
