#!/usr/bin/env python3
#
# Runs a performance test on some tasks to measure parse-time performance.

import json
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
flags.DEFINE_string('output', 'results.json', 'File to write results to')
flags.DEFINE_integer('number', 5, 'Number of times to run test')
FLAGS = flags.FLAGS


def run(i: int):
    """Run once and return the length of time taken."""
    log.info('Run %d of %d', i + 1, FLAGS.number)
    start = time.time()
    subprocess.check_call([FLAGS.plz, 'query', 'alltargets'], stdout=subprocess.DEVNULL)
    duration = time.time() - start
    log.info('Complete in %0.2fs', duration)
    return duration


def main(argv):
    results = [run(i) for i in range(FLAGS.number)]
    median = results[len(results)//2]
    log.info('Complete, median time: %0.2f', median)
    log.info('Running again to generate profiling info')
    subprocess.check_call([FLAGS.plz, 'query', 'alltargets', '--profile_file', 'plz.prof'],
                          stdout=subprocess.DEVNULL)
    log.info('Generating results')
    results.sort()
    with open(FLAGS.output, 'w') as f:
        json.dump({
            'parse': {
                'raw': results,
                'median': median,
            },
        }, f)


if __name__ == '__main__':
    app.run(main)
