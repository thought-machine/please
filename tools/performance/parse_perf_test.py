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
flags.DEFINE_integer('number', 5, 'Number of times to run test', short_name='n')
flags.DEFINE_string('root', 'tree', 'Directory to run in')
FLAGS = flags.FLAGS


def plz() -> list:
    """Returns the plz invocation for a subprocess."""
    return [
        "/usr/bin/time",
        "-f", "%e %M",
        FLAGS.plz,
        '--repo_root', FLAGS.root,
        '--num_threads', str(FLAGS.num_threads),
        'query', 'alltargets',
    ]


def run(i: int):
    """Run once and return the length of time taken."""
    log.info('Run %d of %d', i + 1, FLAGS.number)
    try :
        duration, mem = parse_time_output(subprocess.run(plz(), check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE).stderr.decode("utf-8"))
    except subprocess.CalledProcessError as err:
        log.exception('Subprocess failed: ' + err.stderr.decode())
        raise
    log.info('Complete in %0.2fs, using %d KB', duration, mem)
    return duration, mem


def parse_time_output(output):
    parts = output.split(" ")
    return float(parts[0].strip()), int(parts[1].strip())


def read_cpu_info():
    """Return the CPU model number & number of CPUs."""
    try:
        with open('/proc/cpuinfo') as f:
            models = [line[line.index(':')+2:] for line in f if line.startswith('model name')]
        return models[0].strip(), len(models)
    except:
        log.exception('Failed to read CPU info')
        return '', 0


def main(argv):
    FLAGS.root = os.path.abspath(FLAGS.root)
    results = [run(i) for i in range(FLAGS.number)]

    time_results = [time for time, _ in results ]
    mem_results = [mem for _, mem in results ]

    time_results.sort()
    median_time = time_results[len(time_results)//2]

    mem_results.sort()
    median_mem = mem_results[len(mem_results)//2]

    log.info('Complete, median time: %0.2fs, median mem: %0.2f KB', median_time, median_mem)
    log.info('Running again to generate profiling info')
    profile_file = os.path.join(os.getcwd(), 'plz.prof')
    mem_profile_file = os.path.join(os.getcwd(), 'plz_mem.prof')
    subprocess.check_call(plz() + ['--profile_file', profile_file, '--mem_profile_file', mem_profile_file], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    log.info('Reading CPU info')
    cpu_model, num_cpus = read_cpu_info()
    log.info('Generating results')
    with open(FLAGS.output, 'w') as f:
        json.dump({
            'revision': FLAGS.revision,
            'timestamp': datetime.datetime.now().isoformat(),
            'parse': {
                'raw': time_results,
                'median': median_time,
            },
            'mem': {
                'raw': mem_results,
                'median': median_mem,
            },
            'cpu': {
                'model': cpu_model,
                'num': num_cpus,
            },
        }, f)
        f.write('\n')


if __name__ == '__main__':
    app.run(main)
