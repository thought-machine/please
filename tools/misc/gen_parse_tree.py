#!/usr/bin/env python3
#
# Script to generate a big tree of BUILD files to measure parsing performance.
# Accordingly, it doesn't create any corresponding source files.

import os
import random
import shutil
import subprocess
from math import log10

from third_party.python import colorlog
from third_party.python.absl import app, flags

handler = colorlog.StreamHandler()
handler.setFormatter(colorlog.ColoredFormatter('%(log_color)s%(levelname)s: %(message)s'))
log = colorlog.getLogger(__name__)
log.addHandler(handler)
log.propagate = False  # Needed to stop double logging?

flags.DEFINE_integer('size', 100000, 'Number of BUILD files to generate')
flags.DEFINE_integer('seed', 42, 'Random seed')
flags.DEFINE_string('root', 'tree', 'Directory to put all files under')
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


def main(argv):
    # Ensure this is deterministic
    random.seed(FLAGS.seed)
    packages = []
    pkgset = set()
    filenames = []
    shutil.rmtree(FLAGS.root)
    for i in range(FLAGS.size):
        depth = random.randint(1, 1 + log10(FLAGS.size))
        dir = '/'.join([FLAGS.root] + [random.choice(DIRNAMES) for _ in range(depth)])
        if dir in pkgset:
            continue
        os.makedirs(dir, exist_ok=True)
        base = os.path.basename(dir)
        filename = os.path.join(dir, 'BUILD')
        with open(filename, 'w') as f:
            lang = random.choice(LANGUAGES)
            f.write(LANGUAGE_TEMPLATE.format(
                name = base,
                lang = lang,
                ext = LANGUAGE_EXTENSIONS[lang],
                deps = choose_deps(packages),
                test_deps = [':' + base] + choose_deps(packages) + TEST_DEPS[lang],
            ))
        packages.append(dir)
        pkgset.add(dir)
        filenames.append(filename)
    # Format them all up
    subprocess.check_call(['plz', 'fmt', '-w'] + filenames)


def choose_deps(candidates:list) -> list:
    """Chooses a set of dependencies from the given list."""
    if not candidates:
        return []
    n = random.randint(0, min(len(candidates), 10))
    label = lambda x: f'//{x}:{os.path.basename(x)}'
    return [label(random.choice(candidates)) for _ in range(n)]




if __name__ == '__main__':
    app.run(main)
