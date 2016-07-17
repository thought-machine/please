"""Build rules for compiling protocol buffers & gRPC service stubs.

Note that these are some of the most complex of our built-in build rules,
because of their cross-language nature. Each proto_library rule declares a set of
sub-rules to run protoc & the appropriate java_library, go_library rules etc. Users
shouldn't worry about those sub-rules and just declare a dependency directly on
the proto_library rule to get its appropriate outputs.
"""

# Languages to generate protos for.
# We maintain this internal mapping to normalize their names to the same ones we use elsewhere.
_PROTO_LANGUAGES = {'cc': 'cpp', 'cc_hdrs': 'cpp', 'py': 'python', 'java': 'java', 'go': 'go'}
# File extensions that are produced for each language.
_PROTO_FILE_EXTENSIONS = {
    'cc': ['.pb.cc'],
    'cc_hdrs': ['.pb.h'],
    'py': ['_pb2.py'],
    'go': ['.pb.go'],
    'java': ['.java'],
}


def proto_library(name, srcs, plugins=None, deps=None, visibility=None, labels=None,
                  python_deps=None, cc_deps=None, java_deps=None, go_deps=None,
                  protoc_version=CONFIG.PROTOC_VERSION, languages=None):
    """Compile a .proto file to generated code for various languages.

    Args:
      name (str): Name of the rule
      srcs (list): Input .proto files.
      plugins (dict): Plugins to invoke for code generation.
      deps (list): Dependencies
      visibility (list): Visibility specification for the rule.
      labels (list): List of labels to apply to this rule.
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
    labels = labels or []
    plugins = plugins or {}
    python_deps = python_deps or []
    cc_deps = cc_deps or []
    java_deps = java_deps or []
    go_deps = go_deps or []
    deps = deps or []

    # We detect output names for normal sources, but will have to do a post-build rule for
    # any input rules. We could just do that for everything but it's nicer to avoid them
    # when possible since they obscure what's going on with the build graph.
    file_srcs = [src for src in srcs if src[0] not in {':', '/'}]
    find_outs = lambda lang, ext: [src.replace('.proto', ext) for src in file_srcs] if lang in languages else []
    outs = {
        'py': find_outs('py', '_pb2.py'),
        'go': find_outs('go', '.pb.go'),
        'cc': find_outs('cc', '.pb.cc'),
        'cc_hdrs': find_outs('cc', '.pb.h'),
        'grpc_py': find_outs('py', '_pb2.py'),
        'grpc_go': find_outs('go', '.pb.go'),
        'grpc_cc': find_outs('cc', '.pb.cc') + find_outs('cc', '.grpc.pb.cc'),
        'grpc_cc_hdrs': find_outs('cc', '.pb.h') + find_outs('cc', '.grpc.pb.h'),
    }
    need_post_build = file_srcs != srcs
    provides = {'proto': ':_%s#proto' % name}

    # Used to collect output files for Java we don't know where they'll end up because their
    # location depends on the java_package option defined in the .proto file.
    # For other language we might not know if the sources come from another rule.
    def _annotate_outs(language, extensions):
        def _annotate_outs(rule_name, output):
            for out in output:
                for ext in extensions:
                    if out.endswith(ext):
                        add_out(rule_name, out.lstrip('./'))
        return _annotate_outs

    if 'cc' in languages:
        languages = ['cc_hdrs'] + languages  # Order is important
        plugins['cc_hdrs'] = plugins.get('cc', [])
    for language in languages:
        gen_name = '_%s#protoc_%s' % (name, language)
        gen_dep = ':' + gen_name
        lang_name = '_%s#%s' % (name, language)

        plugin_cmds = ''
        lang_deps = deps[:]
        lang_plugins = plugins.get(language, [])
        if language == 'go' and not lang_plugins:
            # Go doesn't come by default, so add it here.
            lang_plugins.append('--plugin=protoc-gen-go=' + _plugin(CONFIG.PROTOC_GO_PLUGIN, lang_deps))
        cmds = [
            '%s --%s_out=$TMP_DIR %s ${SRCS}' % (_plugin(CONFIG.PROTOC_TOOL, lang_deps),
                                                 _PROTO_LANGUAGES[language],
                                                 ' '.join(lang_plugins)),
            'mv -f ${PKG}/* .',
        ]
        if language == 'py' and CONFIG.PROTO_PYTHON_PACKAGE:
            cmds.append('sed -i -e "s/from google.protobuf/from %s/g" *_pb2.py' %
                        CONFIG.PROTO_PYTHON_PACKAGE)
        post_build = None
        if need_post_build or language == 'java':
            cmds.append('find . ' + ' -or '.join('-name "*%s"' % ext
                                                 for ext in _PROTO_FILE_EXTENSIONS[language]))
            post_build = _annotate_outs(language, _PROTO_FILE_EXTENSIONS[language])

        if language == 'go':
            base_path = get_base_path()
            labels += ['proto:go-map: %s/%s=%s/%s' % (base_path, src, base_path, name) for src in srcs
                       if not src.startswith(':') and not src.startswith('/')]

        is_grpc = 'grpc' in protoc_version
        grpc_language = ('grpc_' + language) if is_grpc else language

        cmd = ' && '.join(cmds)
        if protoc_version:
            cmd += ' # protoc v%s' % protoc_version

        build_rule(
            name = gen_name,
            srcs = srcs,
            outs = outs.get(grpc_language),
            cmd = cmd,
            deps = lang_deps,
            requires = ['proto'],
            pre_build = _go_path_mapping(cmd, is_grpc) if language == 'go' else None,
            post_build = post_build,
            labels = labels,
            needs_transitive_deps = True,
            visibility = visibility,
        )
        provides[language] = ':' + lang_name

        if language == 'cc':
            cc_library(
                name = lang_name,
                srcs = [gen_dep],
                hdrs = [':_%s#protoc_cc_hdrs' % name],
                deps = deps + cc_deps,
                visibility = visibility,
                compiler_flags = ['-Wno-unused-parameter'],  # Generated gRPC code is not robust to this.
                pkg_config_libs = ['grpc++', 'grpc', 'protobuf'] if is_grpc else ['protobuf'],
            )
            provides['cc_hdrs'] = ':__%s#cc#hdrs' % name  # Must wire this up by hand

        elif language == 'py':
            python_library(
                name = lang_name,
                srcs = [gen_dep],
                deps = [CONFIG.PROTO_PYTHON_DEP] + deps + python_deps,
                visibility = visibility,
            )

        elif language == 'java':
            java_library(
                name = lang_name,
                srcs = [gen_dep],
                exported_deps = [CONFIG.PROTO_JAVA_DEP] + deps + java_deps,
                visibility = visibility,
            )

        elif language == 'go':
            go_library(
                name = lang_name,
                srcs = [gen_dep],
                out = name + '.a',
                deps = [CONFIG.PROTO_GO_DEP] + deps + go_deps,
                visibility = visibility,
            )
            # Needed for things like cgo_test / cgo_library that need the source in expected places
            build_rule(
                name = '_%s#go_src' % name,
                srcs = [gen_dep],
                outs = [name],
                deps = deps,
                cmd = 'mkdir -p $OUT && cp ${PKG}/*.go $OUT',
                visibility = visibility,
                requires = ['go_src'],
            )
            provides['go_src'] = ':_%s#go_src' % name

    # This simply collects the sources, it's used for other proto_library rules to depend on.
    filegroup(
        name = '_%s#proto' % name,
        srcs = srcs,
        visibility = visibility,
        exported_deps = deps,
        labels = labels,
        requires = ['proto'],
        output_is_complete = False,
    )
    # This is the final rule that directs dependencies to the appropriate language.
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
        plugins['py'] = [
            '--plugin=protoc-gen-grpc-python=' + _plugin(CONFIG.GRPC_PYTHON_PLUGIN, deps),
            '--grpc-python_out=$TMP_DIR',
        ]
        python_deps = (python_deps or []) + [CONFIG.GRPC_PYTHON_DEP]
    if 'java' in languages:
        plugins['java'] = [
            '--plugin=protoc-gen-grpc-java=' + _plugin(CONFIG.GRPC_JAVA_PLUGIN, deps),
            '--grpc-java_out=$TMP_DIR',
        ]
        java_deps = (java_deps or []) + [CONFIG.GRPC_JAVA_DEP]
    if 'go' in languages:
        go_deps = (go_deps or []) + [CONFIG.GRPC_GO_DEP]
    if 'cc' in languages:
        plugins['cc'] = [
            '--plugin=protoc-gen-grpc-cc=' + _plugin(CONFIG.GRPC_CC_PLUGIN, deps),
            '--grpc-cc_out=$TMP_DIR',
        ]
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


def _go_path_mapping(cmd, include_grpc):
    """Used to update the Go path mapping; by default it doesn't really import in the way we want."""
    def _map_go_paths(rule_name):
        mapping = ',M'.join(get_labels(rule_name, 'proto:go-map:'))
        # Bit of a hack, it's very hard to insert this one generically because of the way the
        # go code generator specifies its own plugins.
        grpc_plugin = 'plugins=grpc,' if include_grpc else ''
        new_cmd = cmd.replace('go_out=$TMP_DIR', 'go_out=%sM%s:$TMP_DIR' % (grpc_plugin, mapping))
        set_command(rule_name, new_cmd)
    return _map_go_paths
