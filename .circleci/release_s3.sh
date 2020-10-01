#!/usr/bin/env bash

VERSION=$(cat VERSION)
if s3 ls s3://please-releases/linux_amd64/$VERSION; then
  echo "Please $VERSION has already been released, nothing to do."
  exit 0
fi
echo "Releasing Please $VERSION"

find /tmp/workspace/*_amd64 -type f | xargs /tmp/workspace/release_signer

aws s3 sync /tmp/workspace/darwin_amd64 s3://please-releases/darwin_amd64/$VERSION
aws s3 sync /tmp/workspace/linux_amd64 s3://please-releases/linux_amd64/$VERSION
aws s3 sync /tmp/workspace/freebsd_amd64 s3://please-releases/freebsd_amd64/$VERSION