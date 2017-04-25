#!/bin/bash
# Script to automatically upload new Please versions.
# Will not overwrite existing versions, only adds new ones.
# Should be run from the root of the repo, and only by a CI system.

set -euo pipefail
RED="$(tput setaf 1)"
GREEN="$(tput setaf 2)"
YELLOW="$(tput setaf 3)"
RESET="$(tput sgr0)"
BUCKET="s3://get.please.build"

VERSION="$(cat VERSION)"
eval $(go env)
echo "${GREEN}Identifying outputs...${RESET}"
FILES="$(plz query outputs -p //package:tarballs)"

if [ "$GOOS" == "linux" ]; then
    # For Linux we upload debs as well.
    DEBS="$(plz query alltargets -p //package:all --include deb | plz query outputs -p -)"
    FILES="$FILES $DEBS"
fi

UPLOADED=''
for FILE in $FILES; do
    BN="$(basename $FILE)"
    DEST="${BUCKET}/${GOOS}_${GOARCH}/${VERSION}/${BN}"
    echo "${GREEN}Checking ${DEST}...${RESET}"
    if aws s3 ls $DEST > /dev/null ; then
        echo "${YELLOW}${DEST} already exists, will not update${RESET}"
    else
        echo "${GREEN}Uploading ${FILE} to ${DEST}...${RESET}"
        aws s3 cp $FILE $DEST
        UPLOADED=true
    fi
done

if [[ "$VERSION" =~ .*(alpha|beta|pre|rc).* ]]; then
    echo "${YELLOW}Pre-release version found, will not update latest_version.${RESET}"
elif [ "$UPLOADED" = true ]; then
    echo "${GREEN}Uploaded at least one file, updating latest_version...${RESET}"
    aws s3 cp "VERSION" "${BUCKET}/latest_version"
else
    echo "${YELLOW}Didn't upload anything, will not update latest_version${RESET}"
fi
