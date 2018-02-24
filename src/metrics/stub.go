// +build bootstrap

// Used at initial bootstrap only so we don't depend on Prometheus for that.

package metrics

import "core"
import "time"

// InitFromConfig does nothing in this file, it's just a stub.
func InitFromConfig(config *core.Configuration) {}

// Record does nothing in this file, it's just a stub.
func Record(target *core.BuildTarget, d time.Duration) {}

// Stop does nothing in this file, it's just a stub.
func Stop() {}
