package cache

import (
	"github.com/thought-machine/please/src/core"
)

type noopCache struct {

}

func (n *noopCache) Store(target *core.BuildTarget, key []byte, files []string) {

}

func (n *noopCache) Retrieve(target *core.BuildTarget, key []byte, files []string) bool {
	return false
}

func (n *noopCache) Clean(target *core.BuildTarget) {
}

func (n *noopCache) CleanAll() {
}

func (n *noopCache) Shutdown() {
}

