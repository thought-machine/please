""" Rules to build Python code.

The output artifacts for Python rules are .pex files (see https://github.com/pantsbuild/pex).
Pex is a rather nice system for combining Python code and all needed dependencies
(excluding the actual interpreter and possibly some system level bits) into a single file.

The process of compiling pex files can be a little slow when including many large files, as
often happens when one's binary includes large compiled dependencies (eg. numpy...). Hence
we have a fairly elaborate optimisation whereby each python_library rule builds a little
zipfile containing just its sources, and all of those are combined at the end to produce
the final .pex. This builds at roughly the same pace for a clean build of a single target,
but is drastically faster for building many targets with similar dependencies or rebuilding
a target which has only had small changes.
"""


def python_library(name, srcs=None, resources=None, deps=None, visibility=None,
                   test_only=False, zip_safe=True, labels=None,
                   interpreter=CONFIG.DEFAULT_PYTHON_INTERPRETER):
    """Generates a Python library target, which collects Python files for use by dependent rules.

    Note that each python_library performs some pre-zipping of its inputs before they're combined
    in a python_binary or python_test. Hence while it's of course not required that all dependencies
    of those rules are python_library rules, it's often a good idea to wrap any large dependencies
    in one to improve incrementality (not necessary for pip_library, of course).

    Args:
      name (str): Name of the rule.
      srcs (list): Python source files for this rule.
      resources (list): Non-Python files that this rule collects which will be included in the final .pex.
                        The distinction between this and srcs is fairly arbitrary and historical, but
                        semantically quite nice and parallels python_test.
      deps (list): Dependencies of this rule.
      visibility (list): Visibility specification.
      test_only (bool): If True, can only be depended on by tests.
      zip_safe (bool): Should be set to False if this library can't be safely run inside a .pex
                       (the most obvious reason not is when it contains .so modules).
                       See python_binary for more information.
      labels (list): Labels to apply to this rule.
      interpreter (str): The Python interpreter to use. Defaults to the config setting
                         which is normally just 'python', but could be 'python3' or
                         'pypy' or whatever.
    """
    all_srcs = (srcs or []) + (resources or [])
    deps = deps or []
    labels = labels or []
    if not zip_safe:
        labels.append('py:zip-unsafe')
    if all_srcs:
        pex_tool, tools = _tool_path(CONFIG.PEX_TOOL)
        jarcat_tool, tools = _tool_path(CONFIG.JARCAT_TOOL, tools)
        # Pre-zip the files for later collection by python_binary.
        build_rule(
            name='_%s#zip' % name,
            srcs=all_srcs,
            outs=['.%s.pex.zip' % name],
            cmd='%s --compile && %s -d -o ${OUTS} -i .' % (pex_tool, jarcat_tool),
            building_description='Compressing...',
            requires=['py'],
            test_only=test_only,
            output_is_complete=True,
            tools=tools,
        )
        deps.append(':_%s#zip' % name)

    filegroup(
        name=name,
        srcs=all_srcs,
        deps=deps,
        visibility=visibility,
        output_is_complete=False,
        requires=['py'],
        test_only=test_only,
        labels=labels,
    )


def python_binary(name, main, out=None, deps=None, visibility=None, zip_safe=None,
                  interpreter=CONFIG.DEFAULT_PYTHON_INTERPRETER, labels=None):
    """Generates a Python binary target.

    This compiles all source files together into a single .pex file which can
    be easily copied or deployed. The construction of the .pex is done in parts
    by the dependent python_library rules, and this rule simply builds the
    metadata for it and concatenates them all together.

    Args:
      name (str): Name of the rule.
      main (str): Python file which is the entry point and __main__ module.
      out (str): Name of the output file. Default to name + .pex
      deps (list): Dependencies of this rule.
      visibility (list): Visibility specification.
      zip_safe (bool): Allows overriding whether the output is marked zip safe or not.
                       If set to explicitly True or False, the output will be marked
                       appropriately; by default it will be safe unless any of the
                       transitive dependencies are themselves marked as not zip-safe.
      interpreter (str): The Python interpreter to use. Defaults to the config setting
                         which is normally just 'python', but could be 'python3' or
                         'pypy' or whatever.
      labels (list): Labels to apply to this rule.
    """
    pex_tool, tools = _tool_path(CONFIG.PEX_TOOL)
    jarcat_tool, tools = _tool_path(CONFIG.JARCAT_TOOL, tools)
    deps = deps or []
    cmd = ' '.join([
        'rm -f $SRCS &&',
        pex_tool,
        '--src_dir=${TMP_DIR}',
        '--out=temp.pex',
        '--entry_point=$SRCS',
        '--noscan',
        '--interpreter=' + interpreter,
        '--module_dir=' + CONFIG.PYTHON_MODULE_DIR,
        '--zip_safe',
        # Run it through jarcat to normalise the timestamps.
        '&& %s -i temp.pex -o $OUT --suffix=.pex --preamble="`head -n 1 temp.pex`"' % jarcat_tool,
    ])
    pre_build, cmd = _handle_zip_safe(cmd, zip_safe)

    python_library(
        name='_%s#lib' % name,
        srcs=[main],
        deps=deps,
        visibility=visibility,
    )

    # Use the pex tool to compress the entry point & add all the bootstrap helpers etc.
    build_rule(
        name='_%s#pex' % name,
        srcs=[main],
        outs=['.%s_main.pex.zip' % name],  # just call it .zip so everything has the same extension
        cmd=cmd,
        requires=['py', 'pex'],
        pre_build=pre_build,
        deps=deps,
        needs_transitive_deps=True,  # Needed so we can find anything with zip_safe=False on it.
        tools=tools,
    )
    # This rule concatenates the .pex with all the other precompiled zip files from dependent rules.
    build_rule(
        name=name,
        deps=[':_%s#pex' % name, ':_%s#lib' % name],
        outs=[out or (name + '.pex')],
        cmd=_python_binary_cmds(name, jarcat_tool),
        needs_transitive_deps=True,
        binary=True,
        output_is_complete=True,
        building_description="Creating pex...",
        visibility=visibility,
        requires=['py'],
        tools=tools,
        # This makes the python_library rule the dependency for other python_library or
        # python_test rules that try to import it. Does mean that they cannot collect a .pex
        # by depending directly on the rule, they'll just get the Python files instead.
        # This is not a common case anyway; more usually you'd treat that as a runtime data
        # file rather than trying to pack into a pex. Can be worked around with an
        # intermediary filegroup rule if really needed.
        provides={'py': ':_%s#lib' % name},
        labels=labels,
    )


def python_test(name, srcs, data=None, resources=None, deps=None, labels=None, size=None,
                visibility=None, container=False, timeout=0, flaky=0, test_outputs=None,
                zip_safe=None, interpreter=CONFIG.DEFAULT_PYTHON_INTERPRETER):
    """Generates a Python test target.

    This works very similarly to python_binary; it is also a single .pex file
    which is run to execute the tests. The tests are run via unittest.

    Args:
      name (str): Name of the rule.
      srcs (list): Source files for this test.
      data (list): Runtime data files for the test.
      resources (list): Non-Python files to be included in the pex. Note that the distinction
                        vs. srcs is important here; srcs are passed to unittest for it to run
                        and it may or may not be happy if given non-Python files.
      deps (list): Dependencies of this rule.
      labels (list): Labels for this rule.
      size (str): Test size (enormous, large, medium or small).
      visibility (list): Visibility specification.
      container (bool | dict): If True, the test will be run in a container (eg. Docker).
      timeout (int): Maximum time this test is allowed to run for, in seconds.
      flaky (int | bool): True to mark this test as flaky, or an integer for a number of reruns.
      test_outputs (list): Extra test output files to generate from this test.
      zip_safe (bool): Allows overriding whether the output is marked zip safe or not.
                       If set to explicitly True or False, the output will be marked
                       appropriately; by default it will be safe unless any of the
                       transitive dependencies are themselves marked as not zip-safe.
      interpreter (str): The Python interpreter to use. Defaults to the config setting
                         which is normally just 'python', but could be 'python3' or
                        'pypy' or whatever.
    """
    timeout, labels = _test_size_and_timeout(size, timeout, labels)
    deps = deps or []
    pex_tool, tools = _tool_path(CONFIG.PEX_TOOL)
    jarcat_tool, tools = _tool_path(CONFIG.JARCAT_TOOL, tools)
    cmd=' '.join([
        pex_tool,
        '--src_dir=${TMP_DIR}',
        '--out=temp.pex',
        '--entry_point=test_main',
        '--test_package=${PKG//\//.}',
        '--test_srcs=${SRCS_SRCS// /,}',
        '--interpreter=' + interpreter,
        '--module_dir=' + CONFIG.PYTHON_MODULE_DIR,
        '--zip_safe',
        # Run it through jarcat to normalise the timestamps.
        '&& %s -i temp.pex -o $OUT --suffix=.pex --preamble="`head -n 1 temp.pex`"' % jarcat_tool,
    ])
    pre_build, cmd = _handle_zip_safe(cmd, zip_safe)

    # Used to separate dependencies from the pex rule so they aren't recompressed again.
    build_rule(
        name='_%s#deps' % name,
        deps=deps,
        cmd='true',
        requires=['py'],
        test_only=True,
    )

    # Use the pex tool to compress the entry point & add all the bootstrap helpers etc.
    build_rule(
        name='_%s#pex' % name,
        srcs={
            'srcs': srcs,
            'resources': resources,
        },
        outs=['.%s_main.pex.zip' % name],  # just call it .zip so everything has the same extension
        cmd=cmd,
        requires=['py'],
        test_only=True,
        building_description="Creating pex info...",
        pre_build=pre_build,
        deps=[':_%s#deps' % name],
        tools=tools,
    )
    # This rule concatenates the .pex with all the other precompiled zip files from dependent rules.
    build_rule(
        name=name,
        deps=[':_%s#pex' % name],
        data=data,
        outs=['%s.pex' % name],
        labels=labels or [],
        cmd=_python_binary_cmds(name, jarcat_tool),
        test_cmd = '$(exe :%s)' % name,
        needs_transitive_deps=True,
        output_is_complete=True,
        binary=True,
        test=True,
        container=container,
        building_description="Building pex...",
        visibility=visibility,
        test_timeout=timeout,
        flaky=flaky,
        test_outputs=test_outputs,
        requires=['py'],
        tools=tools,
    )


def pip_library(name, version, hashes=None, package_name=None, outs=None, test_only=False,
                env=None, deps=None, post_install_commands=None, install_subdirectory=False,
                repo=None, use_pypi=None, patch=None, visibility=None, zip_safe=True, licences=None):
    """Provides a build rule for third-party dependencies to be installed by pip.

    Args:
      name (str): Name of the build rule.
      version (str): Specific version of the package to install.
      hashes (list): List of acceptable hashes for this target.
      package_name (str): Name of the pip package to install. Defaults to the same as 'name'.
      outs (list): List of output files / directories. Defaults to [name].
      test_only (bool): If True, can only be used by test rules or other test_only libraries.
      env (dict): Environment variables to provide during pip install, as a dict (or similar).
      deps (list): List of rules this library depends on.
      post_install_commands (list): Commands run after pip install has completed.
      install_subdirectory (bool): Forces the package to install into a subdirectory with this name.
      repo (str): Allows specifying a custom repo to fetch from.
      use_pypi (bool): If True, will check PyPI as well for packages.
      patch (str | list): A patch file or files to be applied after install.
      visibility (list): Visibility declaration for this rule.
      zip_safe (bool): Flag to indicate whether a pex including this rule will be zip-safe.
      licences (list): Licences this rule is subject to. Default attempts to detect from package metadata.
    """
    package_name = '%s==%s' % (package_name or name, version)
    outs = outs or [name]
    install_deps = []
    post_install_commands = post_install_commands or []
    post_build = None
    use_pypi = CONFIG.USE_PYPI if use_pypi is None else use_pypi
    index_flag = '' if use_pypi else '--no-index'

    repo_flag = ''
    repo = repo or CONFIG.PYTHON_DEFAULT_PIP_REPO
    if repo:
        if repo.startswith('//') or repo.startswith(':'):  # Looks like a build label, not a URL.
            repo_flag = '-f %(location %s)' % repo
            deps.append(repo)
        else:
            repo_flag = '-f ' + repo

    # Environment variables. Must sort in case we were given a dict.
    environment = ' '.join('%s=%s' % (k, v) for k, v in sorted((env or {}).items()))
    target = name if install_subdirectory else '.'

    cmd = '%s install --no-deps --no-compile --no-cache-dir --default-timeout=60 --target=%s' % (CONFIG.PIP_TOOL, target)
    cmd += ' -b build %s %s %s' % (repo_flag, index_flag, package_name)
    cmd += ' && find . -name "*.pyc" -or -name "tests" | xargs rm -rf'

    if not licences:
        cmd += ' && find . -name METADATA -or -name PKG-INFO | grep -v "^./build/" | xargs grep -E "License ?:" | grep -v UNKNOWN | cat'

    if install_subdirectory:
        cmd += ' && rm -rf %s/*.egg-info %s/*.dist-info' % (name, name)

    if patch:
        patches = [patch] if isinstance(patch, str) else patch
        if CONFIG.OS == 'freebsd':
            # --no-backup-if-mismatch is not supported, but we need to get rid of the .orig
            # files for hashes to match correctly.
            cmd += ' && ' + ' && '.join('patch -p0 < $(location %s)' % patch for patch in patches)
            cmd += ' && find . -name "*.orig" | xargs rm'
        else:
            cmd += ' && ' + ' && '.join('patch -p0 --no-backup-if-mismatch < $(location %s)' % patch for patch in patches)

    if post_install_commands:
        cmd = ' && '.join([cmd] + post_install_commands)

    build_rule(
        name = '_%s#install' % name,
        cmd = cmd,
        outs = outs,
        srcs = patches if patch else [],
        deps = install_deps,
        building_description = 'Fetching...',
        hashes = hashes,
        requires=['py'],
        test_only=test_only,
        licences=licences,
        post_build=None if licences else _add_licences,
    )
    # Get this to do the pex pre-zipping stuff.
    python_library(
        name = name,
        srcs = [':_%s#install' % name],
        deps = deps,
        visibility = visibility,
        test_only=test_only,
        zip_safe=zip_safe,
    )


def _handle_zip_safe(cmd, zip_safe):
    """Handles the zip safe flag. Returns a tuple of (pre-build function, new command)."""
    if zip_safe is None:
        return lambda name: (set_command(name, cmd.replace('--zip_safe', ' --nozip_safe'))
                             if has_label(name, 'py:zip-unsafe') else None), cmd
    elif zip_safe:
        return None, cmd
    else:
        return None, cmd.replace('--zip_safe', ' --nozip_safe')


def _add_licences(name, output):
    """Annotates a pip_library rule with detected licences after download."""
    for line in output:
        if line.startswith('License: '):
            for licence in line[9:].split(' or '):  # Some are defined this way (eg. "PSF or ZPL")
                add_licence(name, licence)
            return
        elif line.startswith('Classifier: License'):
            # Oddly quite a few packages seem to have UNKNOWN for the licence but this Classifier
            # section still seems to know what they are licenced as.
            add_licence(name, line.split(' :: ')[-1])
            return
    log.warning('No licence found for %s, should add licences = [...] to the rule',
                name.lstrip('_').split('#')[0])


def _python_binary_cmds(name, jarcat_tool):
    """Returns the commands to use for python_binary and python_test rules."""
    cmd = ' && '.join([
        'PREAMBLE=`head -n 1 $(location :_%s#pex)`' % name,
        '%s -i . -o $OUTS --suffix=.pex.zip --preamble="$PREAMBLE" --include_other --add_init_py --strict' % jarcat_tool,
    ])
    return {
        'opt': cmd,
        'stripped': cmd + ' -e .py -x "*.py"',
    }


if CONFIG.BAZEL_COMPATIBILITY:
    py_library = python_library
    py_binary = python_binary
    py_test = python_test
