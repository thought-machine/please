// This is similar to cmap_targets.go.
// TODO(peterebden, jpoole): When we get generics in 1.18 this would be a prime use case for them.

package core

import (
	"sync"

	"github.com/OneOfOne/cmap/hashers"
)

// packageMap is a concurrent safe sharded map to scale on multiple cores.
// It's a fully specialised version of cmap.CMap for our most commonly used types.
type packageMap struct {
	shards []*packageLMap
}

// newPackageMap creates a new packageMap.
func newPackageMap() *packageMap {
	cm := &packageMap{
		shards: make([]*packageLMap, shardCount),
	}
	for i := range cm.shards {
		cm.shards[i] = newPackageLMapSize(shardCount)
	}
	return cm
}

// Set is the equivalent of `map[key] = val`.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (cm *packageMap) Set(key packageKey, pkg *Package) bool {
	h := hashPackage(key)
	return cm.shards[h&shardMask].Set(key, pkg)
}

// Get is the equivalent of `val, ok := map[key]`.
func (cm *packageMap) Get(key packageKey) (val *Package, ok bool) {
	h := hashPackage(key)
	return cm.shards[h&shardMask].Get(key)
}

// Await waits for the item with the given key to be added to the map and returns it when so.
// It's basically like a blocking Get.
func (cm *packageMap) Await(key packageKey) *Package {
	h := hashPackage(key)
	return cm.shards[h&shardMask].Await(key)
}

// Values returns a slice of all the current values in the map.
// This is a view that an observer could potentially have had at some point around the calling of this function,
// but no particular consistency guarantees are made.
func (cm *packageMap) Values() []*Package {
	ret := []*Package{}
	for _, lm := range cm.shards {
		ret = append(ret, lm.Values()...)
	}
	return ret
}

func hashPackage(key packageKey) uint32 {
	return hashers.Fnv32(key.Subrepo) ^ hashers.Fnv32(key.Name)
}

// A packagePair represents a build package & an awaitable channel for one to exist.
type packagePair struct {
	Package *Package
	Wait    chan struct{}
}

// packageLMap is a simple sync.RWMutex locked map.
// Used by packageMap internally for sharding.
type packageLMap struct {
	m map[packageKey]packagePair
	l sync.RWMutex
}

// newPackageLMapSize is the equivalent of `m := make(map[BuildLabel]packagePair, cap)`
func newPackageLMapSize(cap int) *packageLMap {
	return &packageLMap{
		m: make(map[packageKey]packagePair, cap),
	}
}

// Set is the equivalent of `map[key] = val`.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (lm *packageLMap) Set(key packageKey, pkg *Package) bool {
	lm.l.Lock()
	defer lm.l.Unlock()
	if existing, present := lm.m[key]; present {
		if existing.Package != nil {
			return false  // already added
		}
		// Hasn't been added, but something is waiting for it to be.
		lm.m[key] = packagePair{
			Package: pkg,
			Wait:    existing.Wait,
		}
		if existing.Wait != nil {
			close(existing.Wait)
		}
		return true
	}
	lm.m[key] = packagePair{Package: pkg}
	return true
}

// Get is the equivalent of `val, ok := map[key]`.
func (lm *packageLMap) Get(key packageKey) (*Package, bool) {
	lm.l.RLock()
	defer lm.l.RUnlock()
	v, ok := lm.m[key]
	return v.Package, ok
}

// Await is like a blocking Get.
func (lm *packageLMap) Await(key packageKey) *Package {
	lm.l.Lock()
	v, ok := lm.m[key]
	if !ok || v.Wait == nil {
		v.Wait = make(chan struct{})
		lm.m[key] = v
	}
	lm.l.Unlock()
	<-v.Wait
	t, _ := lm.Get(key)
	return t
}

// Values returns a copy of all the packages currently in the map.
func (lm *packageLMap) Values() []*Package {
	lm.l.RLock()
	defer lm.l.RUnlock()
	ret := make([]*Package, 0, len(lm.m))
	for _, v := range lm.m {
		if v.Package != nil {
			ret = append(ret, v.Package)
		}
	}
	return ret
}
