#!/usr/bin/env python3
"""Script to create Github releases & generate release notes."""

import json
import logging
import os
import subprocess
import sys
from third_party.python import requests


class ReleaseGen:

    def __init__(self, version):
        if 'GITHUB_TOKEN' not in os.environ:
            logging.warning('Unless GITHUB_TOKEN is specified, your requests will likely fail')
        self.url = 'https://api.github.com'
        self.session = requests.Session()
        self.session.headers.update({
            'Accept': 'application/vnd.github.v3+json',
            'Authorization': 'token ' + os.environ.get('GITHUB_TOKEN', ''),
        })
        self.version = version or self.get_current_version()
        self.version_name = 'Version ' + self.version

    def get_current_version(self):
        """Loads the current version from the repo."""
        with open('VERSION') as f:
            return f.read().strip()

    def get_latest_release_version(self):
        """Gets the latest released version from Github."""
        response = self.session.get(self.url + '/repos/thought-machine/please/releases/latest')
        response.raise_for_status()
        return json.loads(response.text).get('tag_name').lstrip('v')

    def release(self):
        """Submits a new release to Github."""
        tag = self.tag_release(self.get_release_sha())
        data = {
            'tag_name': tag,
            'name': 'Please v' + self.version,
            'body': ''.join(self.get_release_notes()),
            'prerelease': 'a' in self.version or 'b' in self.version,
        }
        response = self.session.post(self.url + '/repos/thought-machine/please/releases', json=data)
        print(response.text)
        response.raise_for_status()

    def tag_release(self, commit_sha):
        """Tags the release at the given commit (& matching date)."""
        tag = 'v' + self.version
        # Have to tag at the correct time.
        commit_date = subprocess.check_output(['git', 'show', '-s', '--format=%ci', commit_sha]).decode('utf-8').strip()
        env = os.environ.copy()
        env['GIT_COMMITTER_DATE'] = commit_date
        subprocess.check_call(['git', 'tag', '-a', tag, commit_sha, '-m', 'Please ' + tag], env=env)
        subprocess.check_call(['git', 'push', 'origin', tag])
        return tag

    def get_release_sha(self):
        """Retrieves the Git commit corresponding to the given release."""
        for line in subprocess.check_output(['git', 'blame', 'ChangeLog']).decode('utf-8').split('\n'):
            if self.version_name in line:
                # Convert short SHA to full one
                return subprocess.check_output(['git', 'rev-parse', line.split(' ')[0]]).decode('utf-8').strip()
        raise Exception("Couldn't determine git sha for " + self.version_name)

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
        raise Exception("Couldn't find release notes for " + self.version_name)


if __name__ == '__main__':
    r = ReleaseGen(sys.argv[1] if len(sys.argv) >= 2 else None)
    r.release()
