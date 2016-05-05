"""Built-in rules to compile Java code."""

# Prefixes of files to exclude when building jars. May need to be configurable.
_JAVA_EXCLUDE_FILES = ','.join([
    'META-INF/LICENSE', 'META-INF/NOTICE', 'META-INF/maven/*', 'META-INF/MANIFEST.MF',
    # Unsign all jars by default, after concatenation the signatures will no longer be valid.
    'META-INF/*.SF', 'META-INF/*.RSA', 'META-INF/*.LIST',
])

_MAVEN_CENTRAL = "https://repo1.maven.org/maven2"
_maven_packages = defaultdict(dict)


def java_library(name, srcs=None, resources=None, resources_root=None, deps=None,
                 exported_deps=None, exports=None, visibility=None, source=None,
                 target=None, test_only=False):
    """Compiles Java source to a .jar which can be collected by other rules.

    Args:
      name (str): Name of the rule
      srcs (list): Java source files to compile for this library
      resources (list): Resources to include in the .jar file
      resources_root (str): Root directory to treat resources relative to; ie. if we are in
                            //project/main/resources and resources_root is project/main then
                            the resources in the .jar will be in the subdirectory 'resources'.
      deps (list): Dependencies of this rule.
      exported_deps (list): Exported dependencies, ie. dependencies that other things depending on this
                            rule will also receive when they're compiling. This is quite important for
                            Java; any dependency that forms part of the public API for your classes
                            should be an exported dependency.
      exports (list): Alias for 'exported_deps'.
      visibility (list): Visibility declaration of this rule.
      source (int): Java source level to compile sources as. Defaults to whatever's set in the config,
                    which itself defaults to 8.
      target (int): Java bytecode level to target after compile. Defaults to whatever's set in the
                    config, which itself defaults to 8.
      test_only (bool): If True, this rule can only be depended on by tests.
    """
    all_srcs = (srcs or []) + (resources or [])
    exported_deps = exported_deps or exports
    if srcs:
        build_rule(
            name=name,
            srcs=all_srcs,
            deps=deps,
            exported_deps=exported_deps,
            outs=[name + '.jar'],
            visibility=visibility,
            cmd=' && '.join([
                'mkdir _tmp _tmp/META-INF',
                '%s -encoding utf8 -source %s -target %s -classpath .:%s -d _tmp -g %s' % (
                    CONFIG.JAVAC_TOOL,
                    source or CONFIG.JAVA_SOURCE_LEVEL,
                    target or CONFIG.JAVA_TARGET_LEVEL,
                    r'`find . -name "*.jar" ! -name "*src.jar" | tr \\\\n :`',
                   ' '.join('$(locations %s)' % src for src in srcs)
                ),
                'mv ${PKG}/%s/* _tmp' % resources_root if resources_root else 'true',
                'find _tmp -name "*.class" | sed -e "s|_tmp/|${PKG} |g" -e "s/\\.class/.java/g"  > _tmp/META-INF/please_sourcemap',
                CONFIG.JAR_TOOL + ' cfM $OUT -C _tmp .',
            ]),
            building_description="Compiling...",
            requires=['java'],
            test_only=test_only,
        )
    elif resources:
        # Can't run javac since there are no java files.
        if resources_root:
            cmd = 'cd ${PKG}/%s && %s -cfM ${OUT} .' % (resources_root, CONFIG.JAR_TOOL)
        else:
            cmd = '%s -cfM ${OUT} ${PKG}' % CONFIG.JAR_TOOL
        build_rule(
            name=name,
            srcs=all_srcs,
            deps=deps,
            exported_deps=exported_deps,
            outs=[name + '.jar'],
            visibility=visibility,
            cmd=cmd,
            building_description="Linking...",
            requires=['java'],
            test_only=test_only,
        )
    else:
        # If input is only jar files (as maven_jar produces in some cases) we simply collect them
        # all up for other rules to use.
        filegroup(
            name=name,
            deps=deps,
            exported_deps=exported_deps,
            visibility=visibility,
            output_is_complete=False,
            requires=['java'],
            test_only=test_only,
        )


def java_binary(name, main_class, srcs=None, deps=None, data=None, visibility=None, jvm_args=None,
                self_executable=False):
    """Compiles a .jar from a set of Java libraries.

    Args:
      name (str): Name of the rule.
      main_class (str): Main class to set in the manifest.
      srcs (list): Source files to compile.
      deps (list): Dependencies of this rule.
      data (list): Runtime data files for this rule.
      visibility (list): Visibility declaration of this rule.
      jvm_args (str): Arguments to pass to the JVM in the run script.
      self_executable (bool): True to make the jar self executable.
    """
    if srcs:
        lib_name = '_%s#lib' % name
        java_library(
            name = lib_name,
            srcs = srcs,
            deps = deps,
        )
        deps = deps or []
        deps.append(':' + lib_name)
    if self_executable:
        cmd, tools = _java_binary_cmd(main_class, jvm_args)
    else:
        # This is essentially a hack to get past some Java things (notably Jersey) failing
        # in subtle ways when the jar has a preamble (srsly...).
        cmd, tools = _jarcat_cmd(main_class)
    build_rule(
        name=name,
        deps=deps,
        data=data,
        outs=[name + '.jar'],
        cmd=cmd,
        needs_transitive_deps=True,
        output_is_complete=True,
        binary=True,
        building_description="Creating jar...",
        requires=['java'],
        visibility=visibility,
        tools=tools,
    )


def java_test(name, srcs, data=None, deps=None, labels=None, visibility=None,
              container=False, timeout=0, flaky=0, test_outputs=None,
              test_package=CONFIG.DEFAULT_TEST_PACKAGE, jvm_args=''):
    """Defines a Java test.

    Args:
      name (str): Name of the rule.
      srcs (list): Java files containing the tests.
      data (list): Runtime data files for this rule.
      deps (list): Dependencies of this rule.
      labels (list): Labels to attach to this test.
      visibility (list): Visibility declaration of this rule.
      container (bool | dict): True to run this test within a container (eg. Docker).
      timeout (int): Maximum length of time, in seconds, to allow this test to run for.
      flaky (int | bool): True to mark this as flaky and automatically rerun.
      test_outputs (list): Extra test output files to generate from this test.
      test_package (str): Java package to scan for test classes to run.
      jvm_args (str): Arguments to pass to the JVM in the run script.
    """
    # It's a bit sucky doing this in two separate steps, but it is
    # at least easy and reuses the existing code.
    java_library(
        name='_%s#lib' % name,
        srcs=srcs,
        deps=deps,
        test_only=True,
        # Deliberately not visible outside this package.
    )
    # As above, would be nicer if we could make the jars self-executing again.
    cmd, tools = _jarcat_cmd('net.thoughtmachine.please.test.TestMain')
    junit_runner, tools = _tool_path(CONFIG.JUNIT_RUNNER, tools)
    cmd = 'ln -s %s . && %s' % (junit_runner, cmd)
    test_cmd = 'java -Dnet.thoughtmachine.please.testpackage=%s %s -jar $(location :%s) ' % (
        test_package, jvm_args, name)
    build_rule(
        name=name,
        cmd=cmd,
        test_cmd=test_cmd,
        data=data,
        outs=[name + '.jar'],
        deps=[':_%s#lib' % name],
        visibility=visibility,
        container=container,
        labels=labels,
        test_timeout=timeout,
        flaky=flaky,
        test_outputs=test_outputs,
        requires=['java'],
        needs_transitive_deps=True,
        output_is_complete=True,
        test=True,
        binary=True,
        building_description="Creating jar...",
        tools=tools,
    )


def maven_jars(name, id, repository=_MAVEN_CENTRAL, exclude=None, hashes=None, combine=False,
               hash=None, deps=None, visibility=None, filename=None, deps_only=False,
               optional=None):
    """Fetches a transitive set of dependencies from Maven.

    Requires post build commands to be allowed for this repo.

    Note that this is still fairly experimental; the interface is unlikely to change much
    but it still has issues with some Maven packages.

    Args:
      name (str): Name of the output rule.
      id (str): Maven id of the artifact (eg. org.junit:junit:4.1.0)
      repository (str): Maven repo to fetch deps from.
      exclude (list): Dependencies to ignore when fetching this one.
      hashes (dict): Map of Maven id -> rule hash for each rule produced.
      combine (bool): If True, we combine all downloaded .jar files into one uberjar.
      hash (string): Hash of final produced .jar. For brevity, implies combine=True.
      deps (list): Labels of dependencies, as usual.
      visibility (list): Visibility label.
      filename (str): Filename we attempt to download. Defaults to standard Maven name.
      deps_only (bool): If True we fetch only dependent rules, not this one itself. Useful for some that
                        have a top-level target as a facade which doesn't have actual code.
      optional (list): List of optional dependencies to fetch. By default we fetch none of them.
    """
    if id.count(':') != 2:
        raise ValueError('Bad Maven id string: %s. Must be in the format group:artifact:id' % id)
    existing_packages = _maven_packages[get_base_path()]
    exclude = exclude or []
    combine = combine or hash
    source_name = '_%s#src' % name

    def get_hash(id, artifact=None):
        if hashes is None:
            return None
        artifact = artifact or id.split(':')[1]
        return hashes.get(id, hashes.get(artifact, '<not given>'))

    def create_maven_deps(_, output):
        for line in output:
            if not line:
                continue
            try:
                group, artifact, version, licences = line.split(':')
            except ValueError:
                group, artifact, version = line.split(':')
                licences = None
            if artifact in exclude:
                continue
            # Deduplicate packages
            existing = existing_packages.get(artifact)
            if existing:
                if existing != '%s:%s:%s' % (group, artifact, version):
                    raise ValueError('Package version clash in maven_jars: got %s, but already have %s' % (line, existing))
            else:
                maven_jar(
                    name=artifact,
                    id=line,
                    repository=repository,
                    hash=get_hash(id, artifact),
                    licences=licences.split('|') if licences else None,
                    # We deliberately don't make this rule visible externally.
                )
            add_exported_dep(name, ':' + artifact)
            if combine:
                add_exported_dep(source_name, ':' + artifact)

    deps = deps or []
    exclusions = ' '.join('-e ' + excl for excl in exclude)
    options = ' '.join('-o ' + option for option in optional) if optional else ''
    please_maven_tool, tools = _tool_path(CONFIG.PLEASE_MAVEN_TOOL)
    build_rule(
        name='_%s#deps' % name,
        cmd='%s -r %s %s %s %s' % (please_maven_tool, repository, id, exclusions, options),
        post_build=create_maven_deps,
        building_description='Finding dependencies...',
        tools=tools,
    )
    if combine:
        download_name = '_%s#download' % name
        maven_jar(
            name=download_name,
            id=id,
            repository=repository,
            hash=get_hash(id),
            deps = deps,
            visibility=visibility,
            filename=filename,
        )
        # Combine the sources into a separate uberjar
        cmd, tools = _jarcat_cmd()
        build_rule(
            name=source_name,
            output_is_complete=True,
            needs_transitive_deps=True,
            building_description="Creating source jar...",
            deps=[':' + download_name, ':_%s#deps' % name] + deps,
            outs=[name + '_src.jar'],
            requires=['java'],
            cmd=cmd + ' -s src.jar -e ""',
            tools=tools,
        )
        build_rule(
            name=name,
            hashes=[hash],
            output_is_complete=True,
            needs_transitive_deps=True,
            building_description="Creating jar...",
            deps=[':' + download_name, ':' + source_name, ':_%s#deps' % name] + deps,
            outs=[name + '.jar'],
            requires=['java'],
            visibility=visibility,
            cmd=cmd,
            tools=tools,
        )
    elif not deps_only:
        maven_jar(
            name=name,
            id=id,
            repository=repository,
            hash=get_hash(id),
            deps = deps + [':_%s#deps' % name],
            visibility=visibility,
            filename=filename,
        )
    else:
        build_rule(
            name=name,
            deps=[':_%s#deps' % name],
            exported_deps=deps,
            cmd='true',  # do nothing!
            visibility=visibility,
            requires=['java'],
        )


def maven_jar(name, id=None, artifact=None, repository=_MAVEN_CENTRAL, hash=None, deps=None,
              visibility=None, filename=None, sources=True, licences=None,
              exclude_paths=None):
    """Fetches a single Java dependency from Maven.

    Args:
      name (str): Name of the output rule.
      id (str): Maven id of the artifact (eg. org.junit:junit:4.1.0)
      artifact (str): Alias for 'id'. Can only pass one of them.
      repository (str): Maven repo to fetch deps from.
      hash (str): Hash for produced rule.
      deps (list): Labels of dependencies, as usual.
      visibility (list): Visibility label.
      filename (str): Filename we attempt to download. Defaults to standard Maven name.
      sources (bool): True to download source jars as well.
      licences (list): Licences this package is subject to.
      exclude_paths (list): Paths to remove from the downloaded .jar.
    """
    if not (bool(id) ^ bool(artifact)):
        raise ValueError('Must pass exactly one of "id" or "artifact" to maven_jar')
    id = id or artifact
    _maven_packages[get_base_path()][name] = id
    # TODO(pebers): Handle exclusions, packages with no source available and packages with no version.
    try:
        group, artifact, version = id.split(':')
    except ValueError:
        group, artifact, version, licence = id.split(':')
        if licence and not licences:
            licences = licence.split('|')
    filename = filename or '%s-%s.jar' % (artifact, version)
    bin_url = '/'.join([
        repository,
        group.replace('.', '/'),
        artifact,
        version,
        filename or '%s-%s.jar' % (artifact, version),
    ])
    src_url = bin_url.replace('.jar', '-sources.jar')  # is this always predictable?
    outs = [name + '.jar']
    cmd = 'echo "Fetching %s..." && curl -f %s -o %s' % (bin_url, bin_url, outs[0])
    if exclude_paths:
        cmd += ' && zip -d %s %s' % (outs[0], ' '.join(exclude_paths))
    if sources:
        outs.append(name + '_src.jar')
        cmd += ' && curl -fsSL %s -o %s' % (src_url, outs[1])
    build_rule(
        name=name,
        outs=outs,
        cmd=cmd,
        hashes=[hash] if hash else None,
        licences=licences,
        exported_deps=deps,  # easiest to assume these are always exported.
        visibility=visibility,
        building_description='Fetching...',
        requires=['java'],
    )


def _java_binary_cmd(main_class, jvm_args, test_package=None):
    """Returns the command we use to build a .jar for a java_binary or a java_test."""
    prop = '-Dnet.thoughtmachine.please.testpackage=' + test_package if test_package else ''
    preamble = '#!/bin/sh\nexec java %s %s -jar $0 $@' % (prop, jvm_args or '')
    jarcat_cmd, tools = _jarcat_cmd(main_class, preamble)
    junit_runner, tools = _tool_path(CONFIG.JUNIT_RUNNER, tools)
    return ('ln -s %s . && %s' % (junit_runner, jarcat_cmd) if test_package else jarcat_cmd), tools


def _jarcat_cmd(main_class=None, preamble=None):
    """Returns the command we'd use to invoke jarcat, and the tool paths required."""
    jarcat_tool, tools = _tool_path(CONFIG.JARCAT_TOOL)
    cmd = '%s -i . -o ${OUTS} --exclude_internal_prefix "%s"' % (jarcat_tool, _JAVA_EXCLUDE_FILES)
    if main_class:
        cmd += ' -m "%s"' % main_class
    if preamble:
        return cmd + " -p '%s'" % preamble, tools
    return cmd, tools


if CONFIG.BAZEL_COMPATIBILITY:
    def java_toolchain(javac=None, source_version=None, target_version=None, **kwargs):
        """Mimics some effort at Bazel compatibility.

        This doesn't really have the same semantics and ignores a bunch of arguments but it
        isn't easy for us to behave the same way that they do.
        """
        package(
            javac_tool = javac,
            java_source_level = source_version,
            java_target_level = target_version,
        )
