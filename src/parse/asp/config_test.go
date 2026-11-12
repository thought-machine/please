package asp

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
)

func TestResolvePluginTargetValues(t *testing.T) {
	values := resolvePluginValue([]string{"//path/to:target"}, "path/to/subrepo")
	assert.Equal(t, []string{"///path/to/subrepo//path/to:target"}, values)

	values = resolvePluginValue([]string{"///path/to/another/plugin//path/to:target"}, "path/to/subrepo")
	assert.Equal(t, []string{"///path/to/another/plugin//path/to:target"}, values)

	values = resolvePluginValue([]string{"///path/to/another/plugin//path/to:target"}, "")
	assert.Equal(t, []string{"///path/to/another/plugin//path/to:target"}, values)

	values = resolvePluginValue([]string{"//path/to:target"}, "")
	assert.Equal(t, []string{"/////path/to:target"}, values)
}

// newPluginConfigTest returns an interpreter with builtins loaded, the host state, and the state of a
// plugin named "foo" with two config fields: BAR (inherited from the host) and BAZ (not inherited).
func newPluginConfigTest(t *testing.T) (*interpreter, *core.BuildState, *core.BuildState) {
	t.Helper()
	state := core.NewDefaultBuildState()
	parser := NewParser(state, nil)
	src, err := rules.ReadAsset("builtins.build_defs")
	require.NoError(t, err)
	parser.MustLoadBuiltins("builtins.build_defs", src)

	pluginState := state.ForSubrepo("foo", false)
	pluginState.RepoConfig = &core.Configuration{}
	pluginState.RepoConfig.PluginDefinition.Name = "foo"
	pluginState.RepoConfig.PluginConfig = map[string]*core.PluginConfigDefinition{
		"bar": {DefaultValue: []string{"default_bar"}, Inherit: true},
		"baz": {DefaultValue: []string{"default_baz"}, Inherit: false},
	}
	return parser.interpreter, state, pluginState
}

// subrepoState returns a subrepo state under the given parent, with the given overrides for the foo plugin.
// With no overrides it looks like a subrepo with no .plzconfig (e.g. a go_repo module).
func subrepoState(parent *core.BuildState, name string, overrides map[string][]string) *core.BuildState {
	s := parent.ForSubrepo(name, false)
	s.RepoConfig = &core.Configuration{} // as Initialise does
	if overrides != nil {
		s.RepoConfig.Plugin = map[string]*core.Plugin{"foo": {ExtraValues: overrides}}
	}
	return s
}

// runWithPluginConfig interprets some code in a fresh package scope with the foo plugin's config loaded,
// the same way interpretAll sets one up.
func runWithPluginConfig(t *testing.T, i *interpreter, pluginState *core.BuildState, code string) (*scope, error) {
	t.Helper()
	s := i.scope.NewPackagedScope(context.Background(), core.NewPackage("test/package"), 1)
	s.config = i.getConfig(s.state).Copy()
	s.Set("CONFIG", s.config)
	i.loadPluginConfig(s, pluginState)
	stmts, err := i.parser.ParseData([]byte(code), "BUILD")
	require.NoError(t, err)
	_, err = i.interpretStatements(s, i.parser.optimise(stmts))
	return s, err
}

// fooConfig returns the foo plugin's config as seen by the given scope.
func fooConfig(t *testing.T, s *scope) pyDict {
	t.Helper()
	d, ok := asDict(s.config.MustGet("FOO"))
	require.True(t, ok, "CONFIG.FOO should be a dict")
	return d
}

func TestPluginConfigCacheSharing(t *testing.T) {
	i, state, pluginState := newPluginConfigTest(t)
	state.RepoConfig.Plugin = map[string]*core.Plugin{
		"foo": {ExtraValues: map[string][]string{"bar": {"host_bar"}, "baz": {"host_baz"}}},
	}
	a := subrepoState(state, "a", nil)
	b := subrepoState(state, "b", nil)
	c := subrepoState(state, "c", map[string][]string{"baz": {"c_baz"}})

	host := i.loadCachedPluginConfig(pluginState, state, "foo")
	cfgA := i.loadCachedPluginConfig(pluginState, a, "foo")
	cfgB := i.loadCachedPluginConfig(pluginState, b, "foo")
	cfgC := i.loadCachedPluginConfig(pluginState, c, "foo")

	assert.Equal(t, pyString("host_bar"), host.pyDict["BAR"])
	assert.Equal(t, pyString("host_baz"), host.pyDict["BAZ"])
	// Subrepos without overrides inherit BAR, but BAZ isn't inherited so gets reset to its default.
	assert.Equal(t, pyString("host_bar"), cfgA.pyDict["BAR"])
	assert.Equal(t, pyString("default_baz"), cfgA.pyDict["BAZ"])
	// A subrepo with its own override gets that.
	assert.Equal(t, pyString("host_bar"), cfgC.pyDict["BAR"])
	assert.Equal(t, pyString("c_baz"), cfgC.pyDict["BAZ"])

	// Override-free siblings share a single dict; the host and the overriding subrepo have their own.
	ptr := func(d pyFrozenDict) uintptr { return reflect.ValueOf(d.pyDict).Pointer() }
	assert.Equal(t, ptr(cfgA), ptr(cfgB))
	assert.NotEqual(t, ptr(cfgA), ptr(host))
	assert.NotEqual(t, ptr(cfgA), ptr(cfgC))
	// And repeated lookups for the same state come from the cache.
	assert.Equal(t, ptr(host), ptr(i.loadCachedPluginConfig(pluginState, state, "foo")))
	assert.Equal(t, ptr(cfgC), ptr(i.loadCachedPluginConfig(pluginState, c, "foo")))
}

func TestPluginConfigPackageOverride(t *testing.T) {
	i, _, pluginState := newPluginConfigTest(t)
	s, err := runWithPluginConfig(t, i, pluginState, `package(foo = {"bar": "overridden"})`)
	require.NoError(t, err)
	assert.Equal(t, pyString("overridden"), fooConfig(t, s)["BAR"])
	assert.Equal(t, pyString("default_baz"), fooConfig(t, s)["BAZ"])

	// The override must not leak into the cached config that other packages get.
	s, err = runWithPluginConfig(t, i, pluginState, ``)
	require.NoError(t, err)
	assert.Equal(t, pyString("default_bar"), fooConfig(t, s)["BAR"])
}

func TestPluginConfigPackageOverrideUnknownKey(t *testing.T) {
	i, _, pluginState := newPluginConfigTest(t)
	_, err := runWithPluginConfig(t, i, pluginState, `package(foo = {"wibble": "x"})`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FOO.WIBBLE is not a known config value")
}

func TestPluginConfigIsImmutable(t *testing.T) {
	i, _, pluginState := newPluginConfigTest(t)
	_, err := runWithPluginConfig(t, i, pluginState, "foo = CONFIG.FOO\nfoo[\"BAR\"] = \"mutated\"")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dict is immutable")

	// Nothing else sees the attempted change.
	s, err := runWithPluginConfig(t, i, pluginState, ``)
	require.NoError(t, err)
	assert.Equal(t, pyString("default_bar"), fooConfig(t, s)["BAR"])
}

func TestPluginConfigBehavesAsDict(t *testing.T) {
	i, _, pluginState := newPluginConfigTest(t)
	s, err := runWithPluginConfig(t, i, pluginState, `
is_dict = isinstance(CONFIG.FOO, dict)
merged = {"QUX": "qux"} | CONFIG.FOO
`)
	require.NoError(t, err)
	assert.Equal(t, True, s.Lookup("is_dict"))
	merged, ok := asDict(s.Lookup("merged"))
	require.True(t, ok)
	assert.Equal(t, pyString("qux"), merged["QUX"])
	assert.Equal(t, pyString("default_bar"), merged["BAR"])
}
