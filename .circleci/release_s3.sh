#!/usr/bin/env bash

set -eu

VERSION=$(cat VERSION)

if s3 ls s3://please-releases/linux_amd64/$VERSION; then
  echo "Please $VERSION has already been released, nothing to do."
  exit 0
fi
echo "Releasing Please $VERSION"

find /tmp/workspace/darwin_amd64 -name "._*" | xargs rm -rf
find /tmp/workspace/*_amd64 -type f | xargs /tmp/workspace/release_signer

aws s3 sync /tmp/workspace/darwin_amd64 s3://please-releases/darwin_amd64/$VERSION
aws s3 sync /tmp/workspace/linux_amd64 s3://please-releases/linux_amd64/$VERSION
aws s3 sync /tmp/workspace/freebsd_amd64 s3://please-releases/freebsd_amd64/$VERSION

aws s3 cp tools/misc/get_plz.sh s3://please-releases/get_plz.sh --content-type text/x-shellscript

if [[ "$VERSION" == *"beta"* ]] || [[ "$VERSION" == *"alpha"* ]]; then
  echo "$VERSION is a prerelease, only setting latest_prerelease_version"
else
  echo "$VERSION is not a prerelease, setting latest_version and latest_prerelease_version"
  aws s3 cp VERSION s3://please-releases/latest_version  --content-type text/plain
fi
aws s3 cp VERSION s3://please-releases/latest_prerelease_version  --content-type text/plain

aws cloudfront create-invalidation --distribution-id $AWS_CF_RELEASE_DIST_ID --paths /latest_version /latest_prerelease_version /get_plz.sh /darwin_amd64/$VERSION /linux_amd64/$VERSION /freebsd_amd64/$VERSION
