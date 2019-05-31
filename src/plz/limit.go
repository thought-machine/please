package plz

import (
	"sync"

	"github.com/thought-machine/please/src/core"
)

// A limiter is responsible for limiting the number of concurrent invocations of a target.
type limiter struct {
	limits map[string]*limit
	mutex  sync.Mutex
}

// A limit describes an individual limit for a label.
type limit struct {
	Label          string
	Limit, Current int
	Done           chan func()
}

// newLimiter creates a new limiter for the current config.
func newLimiter(config *core.Configuration) *limiter {
	l := &limiter{limits: map[string]*limit{}}
	for _, lim := range config.Limit {
		l.limits[lim.Label] = &limit{
			Label: lim.Label,
			Limit: lim.Limit,
			Done:  make(chan func(), 10000),
		}
	}
	return l
}

// ShouldRun returns true if the given target is able to run now.
// If it is the the caller should proceed, if not the task will be re-queued once the limit
// becomes satisfied and the caller should not handle it.
func (l *limiter) ShouldRun(state *core.BuildState, t *core.BuildTarget, tt core.TaskType) bool {
	if lim := l.findLimit(t); lim != nil {
		log.Notice("Putting %s on hold due to max limit of %d %s at a time", t.Label, lim.Limit, lim.Label)
		lim.Done <- func() {
			// We can't start this now, but we can re-queue it.
			state.AddActiveTarget()
			if tt == core.Test {
				state.AddPendingTest(t.Label)
			} else {
				state.AddPendingBuild(t.Label, tt == core.SubincludeBuild)
			}
			// Only now the existing task counts as done
			state.TaskDone(true)
		}
		return false
	}
	return true
}

func (l *limiter) findLimit(t *core.BuildTarget) *limit {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.limits) > 0 {
		for _, label := range t.Labels {
			if lim, present := l.limits[label]; present && lim.Current >= lim.Limit {
				return lim
			}
		}
		// If we get here we're going to go ahead with it, we need to increment all limits.
		// This is done separately from above to handle the case where there are multiple limits
		// and a target fits in the first one but not a later one.
		for _, label := range t.Labels {
			if lim, present := l.limits[label]; present {
				lim.Current++
			}
		}
	}
	return nil
}

// Done lowers all limits for a target once it's complete.
func (l *limiter) Done(t *core.BuildTarget) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.limits) > 0 {
		for _, label := range t.Labels {
			if lim, present := l.limits[label]; present {
				if lim.Current == lim.Limit {
					go func() {
						f := <-lim.Done
						f()
					}()
				}
				lim.Current--
			}
		}
	}
}
