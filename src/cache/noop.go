package cache

import (
	"github.com/thought-machine/please/src/core"
)

type noopCache struct {
}

func (n *noopCache) Store(*core.BuildTarget, []byte, []string) {

}

func (n *noopCache) Retrieve(*core.BuildTarget, []byte, []string) bool {
	return false
}

func (n *noopCache) Clean(*core.BuildTarget) {
}

func (n *noopCache) CleanAll() {
}

func (n *noopCache) Shutdown() {
}
