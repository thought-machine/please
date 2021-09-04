package core

import (
	"regexp"
	"sync"
)

// A DeferredRegexp is like a normal regexp but defers its initialisation until first use.
// This helps avoid spending lots of time initialising things at init time.
//
// The interface mimics the subset of regexp.Regexp that we find relevant right now.
//
// Note that it generally uses MustCompile internally in order to mimic the regexp interface,
// so you only want to use it for static regexes that you know are valid (which is the only real
// use case anyway).
type DeferredRegexp struct {
	Re   string
	once sync.Once
	re   *regexp.Regexp
}

func (dr *DeferredRegexp) init() {
	dr.once.Do(func() {
		dr.re = regexp.MustCompile(dr.Re)
	})
}

func (dr *DeferredRegexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	dr.init()
	return dr.re.ReplaceAllStringFunc(src, repl)
}

func (dr *DeferredRegexp) FindStringSubmatch(s string) []string {
	dr.init()
	return dr.re.FindStringSubmatch(s)
}

func (dr *DeferredRegexp) FindSubmatch(b []byte) [][]byte {
	dr.init()
	return dr.re.FindSubmatch(b)
}
