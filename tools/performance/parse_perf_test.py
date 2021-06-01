#!/usr/bin/env python3
#
# Runs a performance test on some tasks to measure parse-time performance.

import datetime
import json
import os
import subprocess
import time

from third_party.python import colorlog
from third_party.python.absl import app, flags

handler = colorlog.StreamHandler()
handler.setFormatter(colorlog.ColoredFormatter('%(log_color)s%(levelname)s: %(message)s'))
log = colorlog.getLogger(__name__)
log.addHandler(handler)
log.propagate = False  # Needed to stop double logging?

flags.DEFINE_string('plz', 'plz', 'Binary to run to invoke plz')
flags.DEFINE_integer('num_threads', 10, 'Number of parallel threads to give plz')
flags.DEFINE_string('output', 'results.json', 'File to write results to')
flags.DEFINE_string('revision', 'unknown', 'Git revision')
flags.DEFINE_integer('number', 5, 'Number of times to run test')
flags.DEFINE_string('root', 'tree', 'Directory to run in')
FLAGS = flags.FLAGS


def plz() -> list:
    """Returns the plz invocation for a subprocess."""
    return [
        FLAGS.plz,
        '--repo_root', FLAGS.root,
        '--num_threads', str(FLAGS.num_threads),
        'query', 'alltargets',
    ]


def run(i: int):
    """Run once and return the length of time taken."""
    log.info('Run %d of %d', i + 1, FLAGS.number)
    start = time.time()
    subprocess.check_call(plz(), stdout=subprocess.DEVNULL)
    duration = time.time() - start
    log.info('Complete in %0.2fs', duration)
    return duration


def main(argv):
    FLAGS.root = os.path.abspath(FLAGS.root)
    results = [run(i) for i in range(FLAGS.number)]
    results.sort()
    median = results[len(results)//2]
    log.info('Complete, median time: %0.2f', median)
    log.info('Running again to generate profiling info')
    profile_file = os.path.join(os.getcwd(), 'plz.prof')
    subprocess.check_call(plz() + ['--profile_file', profile_file], stdout=subprocess.DEVNULL)
    log.info('Generating results')
    with open(FLAGS.output, 'w') as f:
        json.dump({
            'revision': FLAGS.revision,
            'timestamp': datetime.datetime.now().isoformat(),
            'parse': {
                'raw': results,
                'median': median,
            },
        }, f)
        f.write('\n')


if __name__ == '__main__':
    app.run(main)
