package wait

import (
	"time"

	"github.com/thought-machine/please/src/cli/logging"
)

var log = logging.Log

func WaitOnChan[T any](ch chan T, message string) T {
	start := time.Now()
	for {
		select {
		case v := <-ch:
			return v
		case <-time.After(time.Second * 10):
			{
				log.Debugf("%v (after %v)", message, time.Since(start))
			}
		}
	}
}
