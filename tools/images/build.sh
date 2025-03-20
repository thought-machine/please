#!/bin/bash

set -euvo pipefail

tag=$(date +%Y%m%d)

reporoot=$(plz query reporoot)
images=("alpine" "freebsd_builder" "ubuntu" "ubuntu_alt")

for image in ${images[@]}; do
  cd "$reporoot/tools/images/$image"
  echo "Building $image"
  docker build . --tag "ghcr.io/thought-machine/please_$image:$tag"
  docker push "ghcr.io/thought-machine/please_$image:$tag"
done
