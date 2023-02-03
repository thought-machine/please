#!/usr/bin/env bash

set -eu

VERSION=$(cat VERSION)

# Synchronises a folder with the aws and gcp buckets.
release_folder() {
  local folder=$1
  local path=$2

  aws s3 sync $folder s3://releases-please-build/$path
  gsutil rsync $folder gs://get.please.build/$path
}

# Copies a file to the bucket, optionally setting the content type
release_file() {
  local file=$1
  local path=$2
  local content_type=$3

  if [ -z "$content_type" ]; then
    aws s3 cp $file s3://releases-please-build/$path
    gsutil cp $file gs://get.please.build/$path
  else
    aws s3 cp $file s3://releases-please-build/$path --content-type $content_type
    gsutil cp -h "Content-Type:$content_type" $file gs://get.please.build/$path
  fi
}

# Auth against gcp for cli and
echo $GCLOUD_SERVICE_KEY > $GOOGLE_APPLICATION_CREDENTIALS
echo $GCLOUD_SERVICE_KEY | gcloud auth activate-service-account --key-file=-

echo "Releasing docs website"
tar -xzf /tmp/workspace/deep-docs.tar.gz -C /tmp/workspace && \
  aws s3 sync /tmp/workspace/docs s3://please-docs && \
  gsutil sync /tmp/workspace/docs gs://please.build


if aws s3 ls s3://please-releases/linux_arm64/$VERSION/; then
  echo "Please $VERSION has already been released, nothing to do."
  exit 0
fi
echo "Releasing Please $VERSION"


find /tmp/workspace/darwin_amd64 -name "._*" | xargs rm -rf
find /tmp/workspace/{*_amd64,*_arm64} -type f | xargs /tmp/workspace/release_signer pgp
find /tmp/workspace/{*_amd64,*_arm64} -type f | xargs /tmp/workspace/release_signer kms

release_folder /tmp/workspace/darwin_amd64 darwin_amd64/$VERSION
release_folder /tmp/workspace/darwin_arm64 darwin_arm64/$VERSION
release_folder /tmp/workspace/linux_amd64 linux_amd64/$VERSION
release_folder /tmp/workspace/linux_arm64 linux_arm64/$VERSION
release_folder /tmp/workspace/freebsd_amd64 freebsd_amd64/$VERSION

# Sign the download script with our release key
/tmp/workspace/release_signer pgp -o get_plz.sh.asc -i tools/misc/get_plz.sh
/tmp/workspace/release_signer kms -o get_plz.sh.sig -i tools/misc/get_plz.sh
release_file tools/misc/get_plz.sh s3://please-releases/get_plz.sh text/x-shellscript
release_file get_plz.sh.asc get_plz.sh.asc text/plain
release_file get_plz.sh.sig get_plz.sh.sig application/octet-stream

if [[ "$VERSION" == *"beta"* ]] || [[ "$VERSION" == *"alpha"* ]] || [[ "$VERSION" == *"prerelease"* ]]; then
  echo "$VERSION is a prerelease, only setting latest_prerelease_version"
else
  echo "$VERSION is not a prerelease, setting latest_version and latest_prerelease_version"
  release_file VERSION latest_version text/plain
fi
release_file VERSION latest_prerelease_version text/plain
