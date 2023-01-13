#!/usr/bin/env python3
"""Script to create Github releases & generate release notes."""

import hashlib
import json
import logging
import os
import subprocess
import sys
import zipfile

from third_party.python import colorlog, requests
from third_party.python.absl import app, flags

handler = colorlog.StreamHandler()
handler.setFormatter(colorlog.ColoredFormatter('%(log_color)s%(levelname)s: %(message)s'))
log = colorlog.getLogger(__name__)
log.addHandler(handler)
log.propagate = False  # Needed to stop double logging?g


flags.DEFINE_string('github_token', os.environ.get('GITHUB_TOKEN'), 'Github API token')
flags.DEFINE_string('circleci_token', os.environ.get('CIRCLECI_TOKEN'), 'CircleCI API token')
flags.DEFINE_string('signer', None, 'Release signer binary')
flags.DEFINE_bool('dry_run', False, "Don't actually do the release, just print it.")
flags.mark_flag_as_required('github_token')
FLAGS = flags.FLAGS

gcp_key_name = "gcpkms://projects/tm-please/locations/eur5/keyRings/please-release/cryptoKeys/please-release/cryptoKeyVersions/1"

PRERELEASE_MESSAGE = """
This is a prerelease version of Please. Bugs and partially-finished features may abound.

Caveat usor!
"""


class ReleaseGen:

    def __init__(self, github_token:str, dry_run:bool=False):
        self.url = 'https://api.github.com'
        self.releases_url = self.url + '/repos/thought-machine/please/releases'
        self.upload_url = self.releases_url.replace('api.', 'uploads.') + '/<id>/assets?name='
        self.session = requests.Session()
        self.session.verify = '/etc/ssl/certs/ca-certificates.crt'
        if not dry_run:
            self.session.headers.update({
                'Accept': 'application/vnd.github.v3+json',
                'Authorization': 'token ' + github_token,
            })
        self.version = self.read_file('VERSION').strip()
        self.version_name = 'Version ' + self.version
        self.is_prerelease = '-' in self.version
        self.known_content_types = {
            '.gz': 'application/gzip',
            '.xz': 'application/x-xz',
            '.asc': 'text/plain',
            '.sha256': 'text/plain',
        }

    def needs_release(self):
        """Returns true if the current version is not yet released to Github."""
        url = self.releases_url + '/tags/v' + self.version
        log.info('Checking %s for release...', url)
        response = self.session.get(url)
        return response.status_code == 404

    def release(self):
        """Submits a new release to Github."""
        data = {
            'tag_name': 'v' + self.version,
            'target_commitish': os.environ.get('CIRCLE_SHA1'),
            'name': 'Please v' + self.version,
            'body': '\n'.join(self.get_release_notes()),
            'prerelease': self.is_prerelease,
            'draft': False,
        }
        if FLAGS.dry_run:
            log.info('Would post the following to Github: %s', json.dumps(data, indent=4))
            return
        log.info('Creating release: %s',  json.dumps(data, indent=4))
        response = self.session.post(self.releases_url, json=data)
        response.raise_for_status()
        data = response.json()
        self.upload_url = data['upload_url'].replace('{?name,label}', '?name=')
        log.info('Release id %s created', data['id'])
    
    def artifact_name(self, artifact):
        arch = self._arch(artifact)
        return os.path.basename(artifact).replace(self.version, self.version + '_' + arch)
    
    def upload(self, artifact:str):
        """Uploads the given artifact to the new release."""
        # Artifact names aren't unique between OSs; make them so.
        
        filename = self.artifact_name(artifact)
        _, ext = os.path.splitext(filename)
        content_type = self.known_content_types.get(ext, 'application/octet-stream')
        url = self.upload_url + filename
        if FLAGS.dry_run:
            log.info('Would upload %s to %s as %s', filename, url, content_type)
            return
        log.info('Uploading %s to %s as %s', filename, url, content_type)
        with open(artifact, 'rb') as f:
            self.session.headers.update({'Content-Type': content_type})
            response = self.session.post(url, data=f)
            response.raise_for_status()
        print('%s uploaded' % filename)

    def _arch(self, artifact:str) -> str:
        cpu = 'amd64' if 'amd64' in artifact else 'arm64'
        if 'darwin' in artifact:
            return f'darwin_{cpu}'
        elif 'freebsd' in artifact:
            return f'freebsd_{cpu}'
        return f'linux_{cpu}'

    def sign_pgp(self, artifact:str) -> str:
        """Creates a detached ASCII-armored signature for an artifact."""
        # We expect the PLZ_GPG_KEY and GPG_PASSWORD env vars to be set.
        out = artifact + '.asc'
        if FLAGS.dry_run:
            log.info('Would sign %s into %s', artifact, out)
        else:
            subprocess.check_call([FLAGS.signer, 'pgp', '-o', out, '-i', artifact])
        return out

    def sign_kms(self, artifact:str) -> str:
        """Signs the artifact with the gcp kms key"""
        out = artifact + '.sig'
        if FLAGS.dry_run:
            log.info('Would sign %s into %s', artifact, out)
        else:
            subprocess.check_call([FLAGS.signer, 'kms', '-o', out, '-i', artifact, '-k', gcp_key_name])
        return out

    def checksum(self, artifact:str) -> str:
        """Creates a file containing a sha256 checksum for an artifact."""
        out = artifact + ".sha256"
        with open(artifact, 'rb') as f:
            checksum = hashlib.sha256(f.read()).hexdigest()
        with open(out, 'w') as f:
            basename = self.artifact_name(artifact)
            f.write(f'{checksum}  {basename}\n')
        return out

    def get_release_notes(self):
        """Yields the changelog notes for a given version."""
        found_version = False
        for line in self.read_file('ChangeLog').split('\n'):
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
            log.warning("No release notes found, continuing anyway since it's a prerelease")
            yield PRERELEASE_MESSAGE.strip()
        else:
            raise Exception("Couldn't find release notes for " + self.version_name)

    def read_file(self, filename):
        """Read a file from the .pex."""
        with zipfile.ZipFile(sys.argv[0]) as zf:
            return zf.read(filename).decode('utf-8')

    def trigger_build(self, token, project):
        """Triggers a CircleCI build of a downstream project."""
        response = self.session.post(
            f'https://circleci.com/api/v1.1/project/github/{project}?circle-token={token}'
        )
        response.raise_for_status()


def main(argv):
    r = ReleaseGen(FLAGS.github_token, dry_run=FLAGS.dry_run)
    if not r.needs_release():
        log.info('Current version has already been released, nothing to be done!')
        return

    release_files = argv[1:]

    # Check we can sign the artifacts before trying to create a release.
    release_files += [r.sign_pgp(artifact) for artifact in argv[1:]]
    release_files += [r.sign_kms(artifact) for artifact in argv[1:]]
    release_files += [r.checksum(artifact) for artifact in argv[1:]]
    r.release()
    for file in release_files:
        r.upload(file)
    if FLAGS.circleci_token and not FLAGS.dry_run:
        r.trigger_build(FLAGS.circleci_token, 'thought-machine/homebrew-please')


if __name__ == '__main__':
    app.run(main)
