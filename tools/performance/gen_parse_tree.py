#!/usr/bin/env python3
#
# Script to generate a big tree of BUILD files to measure parsing performance.
# Accordingly, it doesn't create any corresponding source files.

import os
import random
import shutil
import subprocess
import time
from math import log10

from third_party.python.absl import app, flags
from third_party.python.progress.bar import Bar

flags.DEFINE_string('plz', 'plz', 'Binary to run to invoke plz')
flags.DEFINE_integer('size', 100000, 'Number of BUILD files to generate')
flags.DEFINE_integer('seed', 42, 'Random seed')
flags.DEFINE_string('root', 'tree', 'Directory to put all files under')
flags.DEFINE_boolean('format', True, 'Autoformat all the generated files')
flags.DEFINE_boolean('progress', True, 'Display animated progress bars')
FLAGS = flags.FLAGS


# List of 'representative' directory names of the kind of names programmers would use.
DIRNAMES = [
    'src', 'main', 'cmd', 'tools', 'utils', 'common', 'query', 'process', 'update', 'run',
    'build', 'assets', 'frontend', 'backend', 'worker',
]

# We know multiple languages!
LANGUAGE_EXTENSIONS = {
    'python': 'py',
    'go': 'go',
    'java': 'java',
    'cc': 'cc',
}

LANGUAGES = list(LANGUAGE_EXTENSIONS.keys())

# This is a little fiddly but a nice touch of realism: some targets have very high fan-out
TEST_DEPS = {
    'python': [],
    'go': ['//third_party/go:testify'],
    'java': ['//third_party/java:junit', '//third_party/java:hamcrest'],
    'cc': [],
}

LANGUAGE_TEMPLATE = """
subinclude("///{lang}//build_defs:{lang}")

{lang}_library(
    name = "{name}",
    srcs = glob(["*.{ext}"], exclude=["*_test.{ext}"]),
    deps = {deps},
)

{lang}_test(
    name = "{name}_test",
    srcs = glob(["*_test.{ext}"]),
    deps = {test_deps},
)
"""

GO_SRC_TEMPLATE = """
package {dir}
"""

LANGUAGE_SRC_TEMPLATES = {
    'go': GO_SRC_TEMPLATE,
}


def main(argv):
    # Ensure this is deterministic
    random.seed(FLAGS.seed)
    packages = []
    pkgset = set()
    filenames = []
    progress = lambda desc, it: Bar(desc).iter(it) if FLAGS.progress else it
    if os.path.exists(FLAGS.root):
        shutil.rmtree(FLAGS.root)
    for i in progress('Generating files', range(FLAGS.size)):
        depth = random.randint(1, 1 + int(log10(FLAGS.size)))
        relative_dir = '/'.join([random.choice(DIRNAMES) for _ in range(depth)])
        dir = os.path.join(FLAGS.root, relative_dir)
        if dir in pkgset:
            continue
        os.makedirs(dir, exist_ok=True)
        base = os.path.basename(dir)
        filename = os.path.join(dir, 'BUILD.plz')
        lang = random.choice(LANGUAGES)
        with open(filename, 'w') as f:
            f.write(LANGUAGE_TEMPLATE.format(
                name = base,
                lang = lang,
                ext = LANGUAGE_EXTENSIONS[lang],
                deps = choose_deps(packages),
                test_deps = [':' + base] + choose_deps(packages) + TEST_DEPS[lang],
            ))
        src_template = LANGUAGE_SRC_TEMPLATES.get(lang)
        if src_template:
            basedir = os.path.basename(dir)
            ext = LANGUAGE_EXTENSIONS[lang]
            library_filename = os.path.join(dir, f'{basedir}.{ext}')
            with open(library_filename, 'w') as f:
                f.write(src_template.format(dir = relative_dir))
            test_filename = os.path.join(dir, f'{basedir}_test.{ext}')
            with open(test_filename, 'w') as f:
                f.write(src_template.format(dir = relative_dir))
        packages.append(dir)
        pkgset.add(dir)
        filenames.append(filename)
    # Copy these over directly
    shutil.copytree('third_party', os.path.join(FLAGS.root, 'third_party'))
    shutil.copytree('plugins', os.path.join(FLAGS.root, 'plugins'))
    # Same here but don't bring in the dependency on the root BUILD file, that opens a big can of worms.
    os.mkdir(os.path.join(FLAGS.root, 'build_defs'))
    with open('build_defs/BUILD') as fr, open(os.path.join(FLAGS.root, 'build_defs/BUILD'), 'w') as fw:
        fw.write(fr.read().replace('//:version', 'VERSION'))
    shutil.copy('build_defs/multiversion_wheel.build_defs',
                os.path.join(FLAGS.root, 'build_defs/multiversion_wheel.build_defs'))
    # Create the .plzconfig in the new root
    with open(os.path.join(FLAGS.root, '.plzconfig'), 'w') as f:
        f.write("""
[Plugin "java"]
Target = //plugins:java
[Plugin "python"]
Target = //plugins:python
    """)
        pass
    if FLAGS.format:
        # Format them all up (in chunks to avoid 'argument too long')
        n = 1000
        start = time.time()
        for i in progress('Formatting files', range(0, len(filenames), n)):
            subprocess.check_call([FLAGS.plz, 'fmt', '-w'] + filenames[i: i + n])
        end = time.time()
        if FLAGS.progress:
            print('Formatted %d files in %0.2fs' % (len(filenames), end - start))


def choose_deps(candidates:list) -> list:
    """Chooses a set of dependencies from the given list."""
    if not candidates:
        return []
    n = random.randint(0, min(len(candidates), 10))
    trim = lambda x: x[len(FLAGS.root) + 1:] if x.startswith(FLAGS.root) else x
    label = lambda x: f'//{x}:{os.path.basename(x)}'
    return [label(trim(random.choice(candidates))) for _ in range(n)]

if __name__ == '__main__':
    app.run(main)
