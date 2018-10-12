#!/usr/bin/env python3
"""Script to create Github releases & generate release notes."""

import json
import logging
import os
import subprocess
import sys

from third_party.python import colorlog, requests
from third_party.python.absl import app, flags

logging.root.handlers[0].setFormatter(colorlog.ColoredFormatter('%(log_color)s%(levelname)s: %(message)s'))


flags.DEFINE_string('github_token', None, 'Github API token')
flags.DEFINE_string('version', None, 'Version to release for')
flags.DEFINE_bool('dry_run', False, "Don't actually do the release, just print it.")
flags.mark_flag_as_required('github_token')
FLAGS = flags.FLAGS


PRERELEASE_MESSAGE = """
This is a prerelease version of Please. Bugs and partially-finished features may abound.
"""


class ReleaseGen:

    def __init__(self, version:str , github_token:str):
        self.url = 'https://api.github.com'
        self.releases_url = self.url + '/repos/thought-machine/please/releases'
        self.session = requests.Session()
        self.session.verify = '/etc/ssl/certs/ca-certificates.crt'
        self.session.headers.update({
            'Accept': 'application/vnd.github.v3+json',
            'Authorization': 'token ' + github_token,
        })
        self.version = version or self.get_current_version()
        self.version_name = 'Version ' + self.version
        self.is_prerelease = 'a' in self.version or 'b' in self.version
        self.known_content_types = {
            '.gz': 'application/gzip',
            '.xz': 'application/x-xz',
            '.asc': 'text/plain',
        }

    def get_current_version(self):
        """Loads the current version from the repo."""
        with open('VERSION') as f:
            return f.read().strip()

    def get_latest_release_version(self):
        """Gets the latest released version from Github."""
        response = self.session.get(self.releases_url + '/latest')
        response.raise_for_status()
        return json.loads(response.text).get('tag_name').lstrip('v')

    def needs_release(self):
        """Returns true if the current version is not yet released to Github."""
        return self.get_latest_release_version() != self.version

    def release(self):
        """Submits a new release to Github."""
        tag = self.tag_release(self.get_release_sha())
        data = {
            'tag_name': tag,
            'name': 'Please v' + self.version,
            'body': ''.join(self.get_release_notes()),
            'prerelease': self.is_prerelease,
        }
        if FLAGS.dry_run:
            logging.info('Would post the following to Github: %s' % json.dumps(data, indent=4))
            return
        response = self.session.post(self.releases_url, json=data)
        response.raise_for_status()
        data = response.json()
        self.release_id = data['id']

    def upload(self, artifact:str):
        """Uploads the given artifact to the new release."""
        filename = os.path.basename(artifact)
        _, ext = os.path.splitext(filename)
        content_type = self.known_content_types[ext]
        url = '%s/%s/assets?name=%s' % (self.releases_url, self.release_id, filename)
        with open(artifact, 'rb') as f:
            if FLAGS.dry_run:
                logging.info('Would upload %s to %s as %s' % (filename, url, content_type))
                return
            response = self.session.post(url, files={filename: (filename, f, content_type)})
            response.raise_for_status()
        print('%s uploaded' % filename)

    def tag_release(self, commit_sha):
        """Tags the release at the given commit (& matching date)."""
        tag = 'v' + self.version
        if FLAGS.dry_run:
            logging.info('Would tag %s at commit %s' % (tag, commit_sha))
        else:
            subprocess.check_call(['git', 'tag', '-a', tag, commit_sha, '-m', 'Please ' + tag])
            subprocess.check_call(['git', 'push', 'origin', tag])
        return tag

    def get_release_sha(self):
        """Retrieves the Git commit corresponding to the current release."""
        return subprocess.check_output(['git', 'rev-parse', 'HEAD']).decode('utf-8').strip()

    def get_release_notes(self):
        """Yields the changelog notes for a given version."""
        with open('ChangeLog') as f:
            found_version = False
            for line in f:
                if line.startswith(self.version_name):
                    found_version = True
                    yield 'This is Please v%s' % self.version
                elif line.startswith('------'):
                    continue
                elif found_version:
                    if line.startswith('Version '):
                        return
                    elif line.startswith('   '):
                        # Markdown comes out nicer if we remove some of the spacing.
                        line = line[3:]
                    yield line
        if self.is_prerelease:
            logging.warning("No release notes found, continuing anyway since it's a prerelease")
            return PRERELEASE_MESSAGE
        raise Exception("Couldn't find release notes for " + self.version_name)


def main(argv):
    r = ReleaseGen(FLAGS.version, FLAGS.github_token)
    if not r.needs_release():
        logging.info('Current version is latest release, nothing to be done!')
        return
    r.release()
    for artifact in argv[1:]:
        r.upload(artifact)


if __name__ == '__main__':
    app.run(main)
