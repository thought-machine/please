# Dir cache eviction

## Background

Every repo (and every worktree of a repo) has its own `plz-out`, and these can get large. This
is getting worse with agentic workflows, which tend to create lots of worktrees. We can't share
`plz-out` itself between them, because repo-relative paths like `plz-out/gen/...` are baked
into commands, `$(location)` and so on.

The dir cache already gives us most of the sharing we want. It's on by default at
`~/.cache/please`, so every repo on the machine shares it. It also hardlinks files both when
storing and when retrieving (`fs.RecursiveLink` in `dir_cache.go`). An output built in one
worktree and retrieved in another therefore ends up as a single inode on disk, linked from the
cache and from both `plz-outs`.

The weak point is eviction, which is what this document is about.

## Measurements

We used `tools/misc/plz_out_usage.py`, which reports three totals:

- *apparent*: the sum of each tree's own size.
- *actual*: bytes on disk, counting each inode once.
- *ideal*: one copy of each distinct file content.

### Four worktrees building `//src/...` with a shared, initially empty cache

| | apparent | actual | ideal |
|---|---|---|---|
| 1 worktree (master) | 1.82G | 1.54G | 1.40G |
| + master~5 | 3.63G | 2.09G | 1.95G |
| + master~20 | 5.42G | 2.64G | 2.50G |
| + master~50 (different Go version) | 7.17G | 4.11G | 3.56G |

- Hardlinking through the cache already does most of the work: 7.2G apparent becomes 4.1G
  actual. Without the cache it would be about 6G.
- A perfect content-addressed store would only save a further ~14%. That rules out building
  one for now.
- Each extra worktree costs about 0.55G, even 5 commits apart. Almost all of that is content
  that genuinely differs, mostly statically linked Go test binaries that relink whenever
  `src/core` changes. Sharing whole files can't help with that.
- Filesystem compression, reflinks and copy-on-write can't be relied on. ext4 is the most
  common filesystem for our users.

### A real developer cache, after a few weeks of normal use

- The cache was 8.2G against the 10G high water mark. 7.8G of that was hardlinked into some
  `plz-out`, and only ~0.3G was held by the cache alone.
- About 29k entries had been written in the last day, but only 6.4k existed. The cache was
  thrashing: roughly 22k entries were stored and evicted within a day.
- Every live entry had an atime under a day old.
- 25k of the 31.8k `.lock` files belonged to entries that no longer existed.

## Problems with the original cleaner

The original cleaner walked the whole cache at the start of every plz invocation. If the total
size was over the high water mark (10G), it evicted entries in order of directory atime until
the total was under the low water mark (8G).

1. **Hardlinks aren't accounted for.** Evicting an entry that's also linked into a `plz-out`
   frees nothing. It also breaks sharing: the next worktree that needs the entry rebuilds it
   as a fresh, unshared copy. This is what caused the thrashing.
2. **The recency signal is broken.** With `relatime` (the usual default), atime only moves
   about once a day. Retrieving an entry never lists its directory, but the cleaner's own walk
   does. In effect the "LRU" order recorded when the cleaner last walked each entry.
3. **Every invocation cleans, with no coordination.** With several agents running at once,
   each walks the cache and evicts against the same stale total, so they over-evict together.
   A long-running process never cleans again.
4. **The fixed watermarks are hard to tune.** 10G is far too much on some machines and far
   too little on others, and two thresholds is one more knob than anyone wants.
5. **Lock files leak.** Removing an entry leaves its `.lock` file behind.

## Design

Every rule below only counts, or deletes, files that are *unshared*: no hard link to them
exists outside the cache. The filesystem's link count does the reference counting for us.
When a worktree is deleted, the link counts of its files drop, and their cache entries become
evictable automatically.

A file is unshared if its link count is no more than the number of links to it found inside
the cache during the walk. Comparing against the in-cache count, rather than 1, handles a
single file linked into more than one entry. For example, if both a target and a filegroup
that hardlinks its outputs are cached, the file is unshared once the two cache links are
the only ones left.

### Policy

Three rules decide what gets evicted. Each covers a weakness of the others.

1. **Age (the main rule).** Evict entries that haven't been used for more than 7 days,
   whatever the total size. Linked entries are never evicted, so this only affects entries
   that no `plz-out` points at any more: outputs from deleted worktrees, or outputs that a
   later rebuild replaced in the same worktree. The one real cost is that coming back to a
   branch after more than a week means rebuilding instead of restoring. This matches what
   similar tools do: Go's build cache uses 5 days, and Gradle and Cargo also age out
   entries.
2. **A size cap relative to the disk.** Cap unshared bytes at 10% of the filesystem's total
   size by default, using `unix.Statfs`. The low mark is derived from the cap (e.g. 80% of
   it) rather than being a second setting.
3. **A free-space floor.** If free space on the filesystem drops below max(5% of the disk,
   5GB), evict unshared entries least-recently-used first until free space is back above the
   floor or nothing evictable is left. If we run out, log a single warning saying that the
   space is held by `plz-out` directories, not the cache, so `plz clean` in old worktrees is
   the fix.

Within the size and free-space rules, entries are evicted least-recently-used first. Entries
used within 10 minutes of each other are treated as equally recent, and the larger is evicted
first.

### Recency

Recency comes from the entry's mtime, not its atime:

- A successful retrieve sets the mtime to now, but only if it's more than an hour old, so that
  repeated retrieves don't cause constant writes.
- A store gives the entry a fresh mtime anyway.

### Coordination

- Only one process cleans at a time. It takes a non-blocking lock on `<cache>/.clean.lock`,
  and any other process that can't get the lock skips cleaning.
- A timestamp file records when cleaning last ran. With the age rule doing most of the work,
  a full clean about once a day is enough. Cleaning sooner is only needed when the size cap
  or the free-space floor is breached.

### Config

- `DirCacheMaxAge` (new): defaults to 7 days.
- `DirCacheMinFreeSpace` (new): accepts bytes or a percentage. That needs a new config type,
  because `cli.ByteSize` only takes bytes.
- `DirCacheHighWaterMark`: still honoured if set explicitly, overriding the percentage
  default. It now measures unshared bytes only.
- `DirCacheLowWaterMark`: becomes optional.

On small disks, such as CI runners, the percentage defaults give a smaller cache. They can
set explicit values if that matters.

### Rejected options

- **A machine-wide content-addressed store:** the measurements show only ~14% to gain over
  hardlinking through the cache.
- **Reflinks, copy-on-write clones and filesystem compression:** ext4 doesn't support them.
- **Stripping binaries or keeping fewer test outputs:** out of scope. There will be reasons
  people rely on the current behaviour.
- **Cost-aware eviction** (weighting entries by how long they took to build): extra
  complexity for an unclear gain.
- **A persistent index instead of walking the cache:** the walk takes ~0.1s warm on 330k
  files, so it isn't worth the risk of the index going stale or getting corrupted.
- **Free space as the only trigger:** it ties cache behaviour to unrelated disk usage, it
  can't reclaim space held by `plz-out`, and it still needs a threshold to tune.

## Implementation plan

### PR 1: accounting and recency (done)

- Count each entry's cost as the bytes in its unshared files. The total compared against
  the watermarks is the sum of those costs, with each inode counted once. Entries with zero
  cost are never evicted.
- When files are shared between cache entries, space is only treated as freed once the last
  entry holding a file is removed.
- Bump the entry's mtime on retrieve, at most once an hour, and evict by mtime instead of
  atime. This removes the `djherbis/atime` dependency.
- An entry that disappears mid-walk (e.g. another process cleaned it) is skipped instead of
  aborting the whole walk.

On the real cache above, the counted size drops from 8.2G to ~0.3G.

### PR 2: policy

- The 7-day age rule.
- The size cap as a percentage of the disk, with the low mark derived from it.
- The free-space floor, including the one-time warning.
- New config options, documentation in `docs/config.html`, and a ChangeLog entry.

### PR 3: coordination and housekeeping

- One cleaner at a time, using `<cache>/.clean.lock`.
- A timestamp file so cleaning doesn't run more often than needed.
- Delete an entry's `.lock` file along with it, and sweep orphaned lock files. This has to
  allow for another process racing on the lock file. The existing rename-then-delete in
  `cleanPath` means a racing retrieve just misses, harmlessly.

### Possible later work

- Stop copying the Go toolchain sources between `gen/third_party/go/std` and
  `bin/third_party/go/toolchain`; hardlink them instead.
- Stop re-extracting the same third-party modules when a subrepo's key changes.

Both are small wins found by the measurement script.
