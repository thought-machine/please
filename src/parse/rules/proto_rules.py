"""Build rules for compiling protocol buffers & gRPC service stubs.

Note that these are some of the most complex of our built-in build rules,
because of their cross-language nature. Each proto_library rule declares a set of
sub-rules to run protoc & the appropriate java_library, go_library rules etc. Users
shouldn't worry about those sub-rules and just declare a dependency directly on
the proto_library rule to get its appropriate outputs.
"""

# Languages to generate protos for.
# We maintain this internal mapping to normalize their names to the same ones we use elsewhere.
_PROTO_LANGUAGES = {'cc': 'cpp', 'py': 'python', 'java': 'java', 'go': 'go'}


def proto_library(name, srcs, plugins=None, deps=None, visibility=None,
                  python_deps=None, cc_deps=None, java_deps=None, go_deps=None,
                  protoc_version=CONFIG.PROTOC_VERSION, languages=None):
    """Compile a .proto file to generated code for various languages.

    Args:
      name (str): Name of the rule
      srcs (list): Input .proto files.
      plugins (dict): Plugins to invoke for code generation.
      deps (list): Dependencies
      visibility (list): Visibility specification for the rule.
      python_deps (list): Additional deps to add to the python_library rules
      cc_deps (list): Additional deps to add to the cc_library rules
      java_deps (list): Additional deps to add to the java_library rules
      go_deps (list): Additional deps to add to the go_library rules
      protoc_version (str): Version of protoc compiler, used to invalidate build rules when it changes.
      languages (list): List of languages to generate rules for, chosen from the set {cc, py, go, java}.
    """
    if languages:
        for language in languages:
            if language not in _PROTO_LANGUAGES:
                raise ValueError('Unknown language for proto_library: %s' % language)
    else:
        languages = CONFIG.PROTO_LANGUAGES
    plugins = plugins or {}
    plugin_cmds = ''
    deps = deps or []
    if 'go' in languages and 'plugin=protoc-gen-go' not in plugins:
        plugins['plugin=protoc-gen-go'] = _plugin(CONFIG.PROTOC_GO_PLUGIN, deps)
    if plugins:
        if isinstance(plugins, dict):
            plugins = sorted(plugins.items())
        plugin_cmds = ' '.join('--%s=%s' % (k, v) for k, v in plugins)
    python_deps = python_deps or []
    cc_deps = cc_deps or []
    java_deps = java_deps or []
    go_deps = go_deps or []
    python_outs = [src.replace('.proto', '_pb2.py') for src in srcs] if 'py' in languages else []
    go_outs = [src.replace('.proto', '.pb.go') for src in srcs] if 'go' in languages else []
    cc_hdrs = [src.replace('.proto', '.pb.h') for src in srcs] if 'cc' in languages else []
    cc_srcs = [src.replace('.proto', '.pb.cc') for src in srcs] if 'cc' in languages else []
    gen_name = '_%s#protoc' % name
    gen_rule = ':' + gen_name
    java_only_name = '_%s#java_only' % name
    base_path = get_base_path()
    labels = ['proto:go-map: %s/%s=%s/%s' % (base_path, src, base_path, name) for src in srcs]

    # Used to collect Java output files; we don't know where they'll end up because their
    # location depends on the java_package option defined in the .proto file.
    def _annotate_java_outs(rule_name, output):
        for out in output:
            if out and out.endswith('.java'):
                java_file = out.lstrip('./')
                add_out(rule_name, java_file)
                add_out(java_only_name, ':%s:%s' % (gen_name, java_file))

    protoc_out_flags = ' '.join('--%s_out=$TMP_DIR' % _PROTO_LANGUAGES[language]
                                for language in languages)
    cmds = [
        '%s %s %s ${SRCS}' % (_plugin(CONFIG.PROTOC_TOOL, deps), protoc_out_flags, plugin_cmds),
        'mv -f ${PKG}/* .',
        'find . -name "*.java"  # protoc v%s' % protoc_version,
    ]
    if 'py' in languages and CONFIG.PROTO_PYTHON_PACKAGE:
        cmds.insert(2, 'sed -i -e "s/from google.protobuf/from %s/g" *.py' %
                    CONFIG.PROTO_PYTHON_PACKAGE)
    cmd = ' && '.join(cmds)

    # Used to update the Go path mapping; by default it doesn't really import in the way we want.
    def _go_path_mapping(rule_name):
        mapping = ',M'.join(get_labels(rule_name, 'proto:go-map:'))
        # Bit of a hack, it's very hard to insert this one generically because of the way the
        # go code generator specifies its own plugins.
        grpc_plugin = 'plugins=grpc,' if 'grpc' in protoc_version else ''
        new_cmd = cmd.replace('go_out=$TMP_DIR', 'go_out=%sM%s:$TMP_DIR' % (grpc_plugin, mapping))
        set_command(rule_name, new_cmd)

    build_rule(
        name = gen_name,
        srcs = srcs,
        outs = python_outs + go_outs + cc_hdrs + cc_srcs,
        cmd = cmd,
        deps = deps,
        requires = ['proto'],
        pre_build = _go_path_mapping if 'go' in languages else None,
        post_build = _annotate_java_outs if 'java' in languages else None,
        labels = labels,
        needs_transitive_deps = True,
        visibility = visibility,
    )
    filegroup(
        name = '_%s#proto' % name,
        srcs = srcs,
        visibility = visibility,
        exported_deps = deps,
        labels = labels,
        requires = ['proto'],
        output_is_complete = False,
    )

    provides = {x: ':_%s#%s' % (name, x) for x in ['proto'] + list(languages)}

    if 'cc' in languages:
        cc_library(
            name = '_%s#cc' % name,
            srcs = [':%s:%s' % (gen_name, src) for src in cc_srcs],
            hdrs = [':%s:%s' % (gen_name, hdr) for hdr in cc_hdrs],
            deps = [gen_rule] + deps + cc_deps,
            visibility = visibility,
        )
        provides['cc_hdrs'] = ':__%s#cc#hdrs' % name

    if 'py' in languages:
        python_library(
            name = '_%s#py' % name,
            srcs = [':%s:%s' % (gen_name, out) for out in python_outs],
            deps = [CONFIG.PROTO_PYTHON_DEP] + deps + python_deps,
            visibility = visibility,
        )

    if 'java' in languages:
        # Used to reduce to just the Java files.
        # Can't avoid this as we do with the other rules since java_library won't generate what we want
        # for a rule with no srcs. Also can't use a filegroup directly, it's not quite flexible enough.
        build_rule(
            name = java_only_name,
            srcs = [gen_rule],
            cmd = 'echo $(location_pairs %s) | xargs -n 2 ln -s' % gen_rule,
            skip_cache = True,
        )
        java_library(
            name = '_%s#java' % name,
            srcs = [':' + java_only_name],
            deps = deps,
            exported_deps = [CONFIG.PROTO_JAVA_DEP] + java_deps,
            visibility = visibility,
        )

    if 'go' in languages:
        go_library(
            name = '_%s#go' % name,
            srcs = [':%s:%s' % (gen_name, out) for out in go_outs],
            out = name + '.a',
            deps = [CONFIG.PROTO_GO_DEP] + deps + go_deps,
            visibility = visibility,
        )
        # Needed for things like go_test / cgo_library that need the source in expected places
        build_rule(
            name = '_%s#go_src' % name,
            srcs = [':%s:%s' % (gen_name, out) for out in go_outs],
            outs = ['%s/%s' % (name, src.replace('.proto', '.pb.go')) for src in srcs],
            cmd = 'cp ${PKG}/*.go %s' % name,
            visibility = visibility,
            deps = deps,
            requires = ['go_src'],
        )
        provides['go_src'] = ':_%s#go_src' % name

    filegroup(
        name = name,
        deps = provides.values(),
        provides = provides,
        visibility = visibility,
    )


def grpc_library(name, srcs, deps=None, visibility=None, languages=None,
                 python_deps=None, java_deps=None, go_deps=None):
    """Defines a rule for a grpc library.

    Args:
      name (str): Name of the rule
      srcs (list): Input .proto files.
      deps (list): Dependencies (other proto_library rules)
      python_deps (list): Additional deps to add to the python_library rules
      java_deps (list): Additional deps to add to the java_library rules
      go_deps (list): Additional deps to add to the go_library rules
      visibility (list): Visibility specification for the rule.
      languages (list): List of languages to generate rules for, chosen from the set {cc, py, go, java}.
                        At present this will not create any service definitions for C++, but 'cc' is
                        still accepted for forwards compatibility.
    """
    # No plugin for Go, that's handled above since it's merged into the go_out argument.
    deps = deps or []
    languages = languages or CONFIG.PROTO_LANGUAGES
    plugins = {}
    if 'py' in languages:
        plugins['plugin=protoc-gen-grpc-python'] = _plugin(CONFIG.GRPC_PYTHON_PLUGIN, deps)
        plugins['grpc-python_out'] = '$TMP_DIR'
        python_deps = (python_deps or []) + [CONFIG.GRPC_PYTHON_DEP]
    if 'java' in languages:
        plugins['plugin=protoc-gen-grpc-java'] = _plugin(CONFIG.GRPC_JAVA_PLUGIN, deps)
        plugins['grpc-java_out'] = '$TMP_DIR'
        java_deps = (java_deps or []) + [CONFIG.GRPC_JAVA_DEP]
    if 'go' in languages:
        go_deps = (go_deps or []) + [CONFIG.GRPC_GO_DEP]
    proto_library(
        name = name,
        srcs = srcs,
        plugins=plugins,
        deps=deps,
        python_deps=python_deps,
        java_deps=java_deps,
        go_deps=go_deps,
        languages=languages,
        visibility=visibility,
        protoc_version='%s, grpc v%s' % (CONFIG.PROTOC_VERSION, CONFIG.GRPC_VERSION),
    )


def _plugin(plugin, deps):
    """Handles plugins that are build labels by annotating them with $(exe ) and adds to deps."""
    if plugin.startswith('//'):
        deps.append(plugin)
        return '$(exe %s)' % plugin
    return plugin
