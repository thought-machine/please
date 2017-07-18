package maven

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Workiva/go-datastructures/queue"
)

// A Resolver resolves Maven artifacts into specific versions.
// Ultimately we should really be using a proper SAT solver here but we're
// not rushing it in favour of forcing manual disambiguation (and the majority
// of things going wrong are due to Maven madness anyway).
type Resolver struct {
	sync.Mutex
	// Contains all the poms we've fetched.
	// Note that these are keyed by a subset of the artifact struct, so we
	// can do version-independent lookups.
	poms map[unversioned][]*pomXml
	// Reference to a thing that fetches for us.
	fetch *Fetch
	// Task queue that prioritises upcoming tasks.
	tasks *queue.PriorityQueue
	// Count of live tasks.
	liveTasks int64
}

// NewResolver constructs and returns a new Resolver instance.
func NewResolver(f *Fetch) *Resolver {
	return &Resolver{
		poms:  map[unversioned][]*pomXml{},
		fetch: f,
		tasks: queue.NewPriorityQueue(100, false),
	}
}

// Run runs the given number of worker threads until everything is resolved.
func (r *Resolver) Run(artifacts []Artifact, concurrency int) {
	// Kick off original artifacts
	for _, a := range artifacts {
		r.Submit(&pomDependency{Artifact: a})
	}

	// We use this channel as a slightly overblown semaphore; when any one
	// of the goroutines finishes, we're done. At least one will return but
	// not necessarily more than that.
	ch := make(chan bool, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			r.process()
			ch <- true
		}()
	}
	<-ch
}

// Pom returns a pom for an artifact. The version doesn't have to be specified exactly.
// If one doesn't currently exist it will return nil.
func (r *Resolver) Pom(a *Artifact) *pomXml {
	r.Lock()
	defer r.Unlock()
	return r.pom(a)
}

// CreatePom returns a pom for an artifact. If a suitable match doesn't exist, a new one
// will be created. The second return value is true if a new one was created.
func (r *Resolver) CreatePom(a *Artifact) (*pomXml, bool) {
	r.Lock()
	defer r.Unlock()
	if pom := r.pom(a); pom != nil {
		return pom, false
	}
	// Override an empty version with a suggestion if we're going to create it.
	if a.Version == "" {
		a.SetVersion(a.SoftVersion)
	}
	pom := &pomXml{Artifact: *a}
	r.poms[pom.unversioned] = append(r.poms[pom.unversioned], pom)
	return pom, true
}

func (r *Resolver) pom(a *Artifact) *pomXml {
	poms := r.poms[a.unversioned]
	log.Debug("Resolving %s:%s: found %d candidates", a.GroupId, a.ArtifactId, len(poms))
	for _, pom := range poms {
		if pom.SoftVersion != "" && a.SoftVersion == "" {
			// Unfortunately we can't reuse these if we downloaded as a 'soft' version and
			// we have a request for a real one now; otherwise we might end up using the
			// latest because of this 'soft' guess and never fetch the requested one, but
			// that depends on the order we run the two in which isn't always determinate.
			continue
		}
		if a.Version == "" || pom.ParsedVersion.Matches(&a.ParsedVersion) {
			log.Debug("Retrieving pom %s for %s", pom.Artifact, a)
			return pom
		}
	}
	return nil
}

// Submit adds this dependency to the queue for future resolution.
func (r *Resolver) Submit(dep *pomDependency) {
	atomic.AddInt64(&r.liveTasks, 1)
	r.tasks.Put(dep)
}

// process continually reads tasks from the queue and resolves them.
func (r *Resolver) process() {
	for {
		t, err := r.tasks.Get(1)
		if err != nil {
			log.Fatalf("%s", err)
		}
		dep := t[0].(*pomDependency)
		log.Debug("beginning resolution of %s", dep.Artifact)
		dep.Resolve(r.fetch)
		count := atomic.AddInt64(&r.liveTasks, -1)
		log.Debug("processed %s, %d tasks remaining", dep.Artifact, count)
		if count <= 0 {
			log.Debug("all tasks done, stopping")
			break
		}
	}
}

// Mediate performs recursive dependency version mediation for all artifacts.
// Note that this is only done within each artifact, i.e. technically solving
// this properly is a fairly hard problem, and we avoid that by just attempting
// to solve within each individual artifact.
func (r *Resolver) Mediate() {
	// All can be done in parallel
	var wg sync.WaitGroup
	for _, poms := range r.poms {
		if len(poms) > 1 { // clearly unnecessary otherwise
			wg.Add(1)
			go func(poms []*pomXml) {
				r.mediate(poms)
				wg.Done()
			}(poms)
		}
	}
	wg.Wait()
}

func (r *Resolver) mediate(poms []*pomXml) {
	// strip out parents which we don't need to worry about
	nonParents := make([]*pomXml, 0, len(poms))
	for _, pom := range poms {
		if !pom.isParent {
			nonParents = append(nonParents, pom)
		}
	}
	// Mediation might not be needed any more.
	if len(nonParents) < 2 {
		return
	}
	poms = nonParents

	// Sort it so things are deterministic later
	sort.Slice(poms, func(i, j int) bool {
		return poms[i].ParsedVersion.LessThan(&poms[j].ParsedVersion)
	})

	// Reduce these to just hard deps (any soft versions we assumed, so we don't care about)
	hard := make([]*pomXml, 0, len(poms))
	for _, pom := range poms {
		if pom.SoftVersion == "" {
			hard = append(hard, pom)
		}
	}
	if len(hard) == 1 {
		// Only one hard dep, must be that
		r.updateDeps(poms, hard[0])
		return
	} else if len(hard) == 0 {
		// No hard deps, can just use the first one we find
		r.updateDeps(poms, poms[0])
		return
	}
	// Walk over once and calculate the intersection of all required versions
	ver := hard[0].OriginalArtifact.ParsedVersion
	for _, pom := range hard[1:] {
		if !ver.Intersect(&pom.OriginalArtifact.ParsedVersion) {
			// TODO(peterebden): Should really give some more detail here about what we can't satisfy
			log.Fatalf("Unsatisfiable version constraints for %s:%s: %s", pom.GroupId, pom.ArtifactId, strings.Join(r.allVersions(hard), " "))
		}
	}
	// Find the first one that satisfies this version & use that.
	for _, pom := range hard {
		if pom.ParsedVersion.Matches(&ver) {
			r.updateDeps(poms, pom)
			return
		}
	}
	log.Fatalf("Failed to find a suitable version for %s:%s", poms[0].GroupId, poms[0].ArtifactId)
}

// updateDeps updates all the dependencies to point to one particular artifact.
func (r *Resolver) updateDeps(poms []*pomXml, winner *pomXml) {
	for _, pom := range poms {
		for _, dependor := range pom.Dependors {
			for _, dep := range dependor.Dependencies.Dependency {
				if dep.GroupId == winner.GroupId && dep.ArtifactId == winner.ArtifactId {
					dep.Pom = winner
				}
			}
		}
	}
}

// allVersions returns all the version descriptors in the given set of poms.
func (r *Resolver) allVersions(poms []*pomXml) []string {
	ret := make([]string, len(poms))
	for i, pom := range poms {
		ret[i] = pom.OriginalArtifact.ParsedVersion.Raw
	}
	return ret
}
