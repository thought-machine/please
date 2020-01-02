// +build bootstrap

// Only used at initial bootstrap or when used with 'go run' so we don't have to worry
// about proto compilation until that's sorted.

package cache

import (
	"context"
	"fmt"

	"github.com/thought-machine/please/src/core"
)

func newRPCCache(ctx context.Context, config *core.Configuration) (*httpCache, error) {
	return nil, fmt.Errorf("Config specifies RPC cache but it is not compiled")
}
