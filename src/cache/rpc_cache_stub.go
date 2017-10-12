// Only used at initial bootstrap or when used with 'go run' so we don't have to worry
// about proto compilation until that's sorted.

package cache

import (
	"core"
	"fmt"
)

func newRPCCache(config *core.Configuration) (*httpCache, error) {
	return nil, fmt.Errorf("Config specifies RPC cache but it is not compiled")
}
