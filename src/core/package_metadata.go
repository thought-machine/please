package core

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/thought-machine/please/src/cmap"
)

var zeroBuildStatement BuildStatement

// BuildStatement represents the start and end byte positions of a parsed statement in a BUILD file.
type BuildStatement struct {
	Start, End int
}

// Len returns the byte length of the build statement.
func (bs *BuildStatement) Len() int64 {
	return int64(bs.End - bs.Start)
}

// StartPos returns the starting byte position of the build statement.
func (bs *BuildStatement) StartPos() int64 {
	return int64(bs.Start)
}

// hashBuildStatement mixes the Start and End byte coordinates to produce a 64-bit pseudo hash.
// It uses a multiplicative hashing approach with two large, distinct, odd prime numbers
// to ensure that both Start and End values are thoroughly mixed across all bits,
// especially the lower bits used to determine concurrent map shard indices (e.g., hash & 3).
// Since coding standards and style guidelines (such as 4-space indentations, common targets and names)
// can cause start and end positions to align on predictable byte intervals
// and make them less random, simple bit manipulation is highly biased.
func hashBuildStatement(stmt BuildStatement) uint64 {
	return uint64(stmt.Start)*0x9e3779b97f4a7c15 + uint64(stmt.End)*0xbf58476d1ce4e5b9
}

// BuildStatements is a slice of BuildStatement that implements sort.Interface.
type BuildStatements []BuildStatement

func (s BuildStatements) Len() int           { return len(s) }
func (s BuildStatements) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s BuildStatements) Less(i, j int) bool { return s[i].StartPos() < s[j].StartPos() }

// BuildStatementProvider defines a closure that generates new build statements.
// It is used as an argument in PackageMetadata methods to defer evaluation, avoiding
// unnecessary computation when using the no-op implementation.
type BuildStatementProvider func() BuildStatement

// SubincludesLabelProvider defines a closure that generates labels for a subinclude statement.
// It is used as an argument in PackageMetadata methods to defer evaluation, avoiding
// unnecessary computation when using the no-op implementation.
type SubincludesLabelProvider func() BuildLabels

// StatementMetadata represents all parsed metadata associated with a single statement.
//
// Note: This struct does not currently use an explicit read-write mutex for synchronization.
// Please maintains a separation between the write phase, in this case the interpreter phase, where
// package metadata is written sequentially on a single thread per package as guaranteed by
// [*BuildState.SyncParsePackage] and a subsequent read phase where we support concurrency if the
// metadata is treated as static and read-only.
type StatementMetadata struct {
	// Subincludes tracks the direct, package-level subinclude labels that were required for the
	// successful interpretation of this build statement. This allows mapping the statement back
	// to the subincluded files/symbols it depends on. For transitive subincludes, use
	// [BuildGraph.TransitiveSubincludes] instead.
	Subincludes BuildLabels
	// Files tracks the file paths that were required during interpretation of this statement.
	// These do not reference target sources (which are dependencies of build rules), but rather
	// files evaluated directly during interpretation, such as files matched and captured by a
	// glob() action.
	Files []string
	// Targets lists the build targets produced as a result of this statement. Since a single
	// statement (such as a custom build rule, function call, or loop) can define multiple
	// targets, this is represented as a list of build labels.
	Targets BuildLabels
	// IsSubincludeStatement tracks whether this build statement (identified by its position
	// in the BUILD file) is a subinclude() call.
	IsSubincludeStatement bool
}

// PackageMetadata stores metadata about parsed BUILD files, mapping statements and subincludes
// to their respective targets. This supports additional logic for operations such as `plz export`
// but should be disabled for most operations by using the no-op implementation to avoid the overhead.
type PackageMetadata interface {
	// RegisterStatement records a statement of an interpreted BUILD file and its
	// dependencies. Required subincludes identify the required subincluded targets for a successful
	// interpretation of the statement, i.e. targets that provide the required symbols (variables or
	// methods). The files argument identify the files required when interpreting that statement, for
	// example when interpreting a glob() statement, this argument will include files captured by the
	// file globbing action.
	RegisterStatement(stmt BuildStatement, requiredSubincludes BuildLabels, files []string)
	// RegisterTargetStatement records that the given build target was created as a result of the
	// given statement being executed. For statements that generate targets, we expect this method
	// to be called in addition to [PackageMetadata.RegisterStatement].
	// This should only be called for statements in BUILD files.
	RegisterTargetStatement(target BuildLabel, stmtProvider BuildStatementProvider)
	// RegisterSubincludeStatement records the given build statement (provided by stmtProvider)
	// as being a subinclude statement. We expect this method to be called in addition to
	// [PackageMetadata.RegisterStatement]. This should only be called for statements in BUILD files.
	RegisterSubincludeStatement(stmtProvider BuildStatementProvider)
	// FindStatement returns the build statement that was responsible for generating the given target.
	// Returns an error if the target was not found in the recorded metadata.
	FindStatement(target BuildLabel) (BuildStatement, error)
	// FindTargets returns all build targets that were generated by the given build statement.
	// Returns an empty slice if no targets were found for the given statement.
	FindTargets(stmt BuildStatement) BuildLabels
	// FindRequiredSubincludes returns all subinclude labels that were required by the given target.
	// This method will only report the package level (direct) subincludes, make use of
	// [BuildGraph.TransitiveSubincludes] if you want all the required subincludes for one target.
	FindRequiredSubincludes(target BuildLabel) (BuildLabels, error)
	// FindRelatedTargets finds all the targets that are related to the argument. In this context,
	// target relationship is determined by looking for targets generated by the same build statement.
	// The result excludes the target in the argument.
	FindRelatedTargets(target BuildLabel) (BuildLabels, error)
	// FindPackageLevelRequirements finds all the subincluded labels required by the package that
	// are not associated with BuildTarget generation. An example could be a variable declaration
	// that depends on a subincluded value.
	FindPackageLevelRequirements() (BuildLabels, []string)
	// GetSubincludedLabels returns all build labels that were included by the given subinclude statement.
	// Returns the labels or an empty slice if the statement wasn't found.
	GetSubincludedLabels(stmt BuildStatement) BuildLabels
	// IsInterpretedStatement returns true if the statement provided matches a registered build
	// statement, meaning it was interpreted even if it doesn't generate any targets.
	IsInterpretedStatement(stmt BuildStatement) bool
	// Statements returns all build statement and their metadata tracked in the package, sorted by
	// BUILD file order.
	Statements() ([]BuildStatement, []StatementMetadata)
}

// packageMetadataImpl is the canonical implementation of the PackageMetadata interface. It tracks
// the relationships between BUILD file statements, subincludes, and the build targets they define.
//
// Note: this implementation uses sharded concurrent maps [cmap.Map], however writes (interpreter
// phase) are performed on a single-thread per package. Please guarantees that each package's BUILD file
// and its subincludes are interpreted on exactly one parser thread at a time (enforced by
// SyncParsePackage). Because of this guarantee, non-atomic write methods (like the Get-then-Set
// sequence in RegisterStatement) are safe and wont suffer from write-write race conditions.
// Nevertheless we use concurrent maps to support concurrent reads.
type packageMetadataImpl struct {
	// statements maps each build statement (identified by its byte range in a BUILD file)
	// to its StatementMetadata. Refer to [StatementMetadata] for more details but this single
	// mapping tracks the targets produced by the statement, the subincluded labels required for its
	// interpretation, and other information.
	statements *cmap.Map[BuildStatement, *StatementMetadata]
	// targetToStmt serves as a reverse-lookup map, linking each generated BuildLabel
	// back to the specific BuildStatement that declared it. This is useful for tracing back a target
	// to its statement and to find sibling targets generated by the same statement block.
	targetToStmt *cmap.Map[BuildLabel, BuildStatement]
}

func newPackageMetadata() PackageMetadata {
	return &packageMetadataImpl{
		statements:   cmap.New[BuildStatement, *StatementMetadata](cmap.SmallShardCount, hashBuildStatement),
		targetToStmt: cmap.New[BuildLabel, BuildStatement](cmap.SmallShardCount, HashBuildLabel),
	}
}

// RegisterStatement implements [PackageMetadata.RegisterStatement].
func (m *packageMetadataImpl) RegisterStatement(stmt BuildStatement, requiredSubincludes BuildLabels, files []string) {
	sm, _ := m.statements.AddOrGet(stmt, func() *StatementMetadata {
		return &StatementMetadata{}
	})

	if len(requiredSubincludes) > 0 {
		// It's necessary to support subsequent calls to this method due to dynamic subincludes.
		// A function definition can include calls to subinclude() with a dynamic argument, so here
		// we need to support appending to the required subincludes, meaning this method will be
		// called more than once with different arguments. For an example refer to
		// [test/export/test_dynamic_subinclude].
		sm.Subincludes = mergeSlices(sm.Subincludes, requiredSubincludes)
	}
	if len(files) > 0 {
		// Subsequent calls have to be supported for similar reasons to the above, appending the value.
		sm.Files = mergeSlices(sm.Files, files)
	}
}

// mergeSlices merges two slices of any comparable type. It de-duplicates elements using
// slices.Contains so it should be used for small slices. A set/map approach should be preferred for
// larger slices.
func mergeSlices[T comparable](existing []T, newElements []T) []T {
	merged := append([]T(nil), existing...)
	for _, el := range newElements {
		if !slices.Contains(merged, el) {
			merged = append(merged, el)
		}
	}
	return merged
}

// RegisterTargetStatement implements [PackageMetadata.RegisterTargetStatement].
func (m *packageMetadataImpl) RegisterTargetStatement(target BuildLabel, stmtProvider BuildStatementProvider) {
	stmt := stmtProvider()
	sm, _ := m.statements.AddOrGet(stmt, func() *StatementMetadata {
		return &StatementMetadata{}
	})
	sm.Targets = append(sm.Targets, target)
	m.targetToStmt.Set(target, stmt)
}

// RegisterSubincludeStatement implements [PackageMetadata.RegisterSubincludeStatement].
func (m *packageMetadataImpl) RegisterSubincludeStatement(stmtProvider BuildStatementProvider) {
	stmt := stmtProvider()
	sm, _ := m.statements.AddOrGet(stmt, func() *StatementMetadata {
		return &StatementMetadata{}
	})
	sm.IsSubincludeStatement = true
}

// FindStatement implements [PackageMetadata.FindStatement].
func (m *packageMetadataImpl) FindStatement(target BuildLabel) (BuildStatement, error) {
	stmt := m.targetToStmt.Get(target)
	if stmt == zeroBuildStatement {
		return BuildStatement{}, fmt.Errorf("failed to find statement for target %s", target)
	}
	return stmt, nil
}

// FindTargets implements [PackageMetadata.FindTargets].
func (m *packageMetadataImpl) FindTargets(stmt BuildStatement) BuildLabels {
	if sm := m.statements.Get(stmt); sm != nil {
		return sm.Targets
	}
	return nil
}

// FindRequiredSubincludes implements [PackageMetadata.FindRequiredSubincludes].
func (m *packageMetadataImpl) FindRequiredSubincludes(target BuildLabel) (BuildLabels, error) {
	stmt, err := m.FindStatement(target)
	if err != nil {
		return nil, err
	}

	if sm := m.statements.Get(stmt); sm != nil {
		if len(sm.Subincludes) == 0 {
			// Could be empty if no subincluded label is required
			return nil, nil
		}
		return sm.Subincludes, nil
	}

	return nil, nil
}

// FindRelatedTargets implements [PackageMetadata.FindRelatedTargets].
func (m *packageMetadataImpl) FindRelatedTargets(target BuildLabel) (BuildLabels, error) {
	stmt, err := m.FindStatement(target)
	if err != nil {
		return nil, err
	}
	relatedTargets := m.FindTargets(stmt)
	labels := make(BuildLabels, 0, len(relatedTargets)-1) // -1 since we exclude the target argument
	for _, l := range relatedTargets {
		if l != target {
			labels = append(labels, l)
		}
	}
	return labels, nil
}

// FindPackageLevelRequirements implements [PackageMetadata.FindPackageLevelRequirements].
func (m *packageMetadataImpl) FindPackageLevelRequirements() (BuildLabels, []string) {
	requiredSet := LabelSet{}
	filesSet := map[string]struct{}{}

	// The intention is to finds all the subincluded labels required by the package but not used to
	// generate targets. An example could be a variable declaration that depends on a subincluded value.
	// We range over all interpreted statements that require any subincluded target. From those, we
	// filter out the statements that generate targets and any explicit subinclude() statement calls.
	m.statements.Range(func(stmt BuildStatement, sm *StatementMetadata) {
		if len(sm.Targets) == 0 && !sm.IsSubincludeStatement {
			for _, label := range sm.Subincludes {
				requiredSet.Add(label)
			}
			for _, file := range sm.Files {
				filesSet[file] = struct{}{}
			}
		}
	})

	subincludes := slices.Collect(maps.Keys(requiredSet))
	slices.SortFunc(subincludes, BuildLabel.Compare)

	files := slices.Collect(maps.Keys(filesSet))
	slices.Sort(files)

	return subincludes, files
}

// GetSubincludedLabels implements [PackageMetadata.GetSubincludedLabels].
func (m *packageMetadataImpl) GetSubincludedLabels(stmt BuildStatement) BuildLabels {
	// After determining that this is a subincludes statement we can return the required subincludes
	// registered in the general statement mapping.
	if sm := m.statements.Get(stmt); sm != nil && sm.IsSubincludeStatement {
		return sm.Subincludes
	}
	return nil
}

// IsInterpretedStatement implements [PackageMetadata.IsInterpretedStatement].
func (m *packageMetadataImpl) IsInterpretedStatement(stmt BuildStatement) bool {
	return m.statements.Contains(stmt)
}

// Statements implements [PackageMetadata.Statements].
func (m *packageMetadataImpl) Statements() ([]BuildStatement, []StatementMetadata) {
	type pair struct {
		stmt BuildStatement
		meta StatementMetadata
	}
	pairs := make([]pair, 0)
	m.statements.Range(func(stmt BuildStatement, sm *StatementMetadata) {
		pairs = append(pairs, pair{
			stmt: stmt,
			meta: *sm,
		})
	})

	// Sort the pairs to maintain BUILD file order
	slices.SortFunc(pairs, func(i, j pair) int {
		return cmp.Compare(i.stmt.Start, j.stmt.Start)
	})

	stmts := make([]BuildStatement, len(pairs))
	metas := make([]StatementMetadata, len(pairs))
	for i, p := range pairs {
		stmts[i] = p.stmt
		metas[i] = p.meta
	}

	return stmts, metas
}

// noopPackageMetadata implements the PackageMetadata interface with no-op methods. This is the
// default implementation and is used to avoid the overhead of parsing metadata for operations that
// don't depend on it. To ensure correctness and that the no-op implementation is not used
// unintentionally, all methods which attempt to retrieve information from this no-op implementation
// will cause Please to terminate.
type noopPackageMetadata struct{}

func newNoopPackageMetadata() PackageMetadata {
	return &noopPackageMetadata{}
}

// RegisterStatement implements [PackageMetadata.RegisterStatement].
func (n *noopPackageMetadata) RegisterStatement(stmt BuildStatement, requiredSubincludes BuildLabels, files []string) {
}

// RegisterTargetStatement implements [PackageMetadata.RegisterTargetStatement].
func (n *noopPackageMetadata) RegisterTargetStatement(target BuildLabel, stmtProvider BuildStatementProvider) {
}

// RegisterSubincludeStatement implements [PackageMetadata.RegisterSubincludeStatement].
func (n *noopPackageMetadata) RegisterSubincludeStatement(stmtProvider BuildStatementProvider) {
}

// FindStatement implements [PackageMetadata.FindStatement].
func (n *noopPackageMetadata) FindStatement(target BuildLabel) (BuildStatement, error) {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return BuildStatement{}, nil
}

// FindTargets implements [PackageMetadata.FindTargets].
func (n *noopPackageMetadata) FindTargets(stmt BuildStatement) BuildLabels {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return nil
}

// FindRequiredSubincludes implements [PackageMetadata.FindRequiredSubincludes].
func (n *noopPackageMetadata) FindRequiredSubincludes(target BuildLabel) (BuildLabels, error) {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return nil, nil
}

// FindRelatedTargets implements [PackageMetadata.FindRelatedTargets].
func (n *noopPackageMetadata) FindRelatedTargets(target BuildLabel) (BuildLabels, error) {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return nil, nil
}

// FindPackageLevelRequirements implements [PackageMetadata.FindPackageLevelRequirements].
func (n *noopPackageMetadata) FindPackageLevelRequirements() (BuildLabels, []string) {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return nil, nil
}

// GetSubincludedLabels implements [PackageMetadata.GetSubincludedLabels].
func (n *noopPackageMetadata) GetSubincludedLabels(stmt BuildStatement) BuildLabels {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return nil
}

// IsInterpretedStatement implements [PackageMetadata.IsInterpretedStatement].
func (n *noopPackageMetadata) IsInterpretedStatement(stmt BuildStatement) bool {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return false
}

// Statements implements [PackageMetadata.Statements].
func (n *noopPackageMetadata) Statements() ([]BuildStatement, []StatementMetadata) {
	log.Fatalf("metadata not tracked, using no-op implementation")
	return nil, nil
}
