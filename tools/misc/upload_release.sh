#!/bin/bash
# Script to automatically upload new Please versions.
# Will not overwrite existing versions, only adds new ones.
# Should be run from the root of the repo, and only by a CI system.

set -euo pipefail
RED="\x1B[31m"
GREEN="\x1B[32m"
YELLOW="\x1B[33m"
RESET="\x1B[0m"

BUCKET="s3://get.please.build"
PLZ="plz-out/bin/src/please"

VERSION="$(cat VERSION)"
eval $(go env)
echo -e "${GREEN}Identifying outputs...${RESET}"
FILES="$($PLZ query outputs -p //package:tarballs)"

if [ "$GOOS" == "linux" ]; then
    # For Linux we upload debs as well.
    DEBS="$($PLZ query alltargets -p //package:all --include deb | $PLZ query outputs -p -)"
    FILES="$FILES $DEBS"
fi

UPLOADED=''
for FILE in $FILES; do
    BN="$(basename $FILE)"
    DEST="${BUCKET}/${GOOS}_${GOARCH}/${VERSION}/${BN}"
    echo -e "${GREEN}Checking ${DEST}...${RESET}"
    if aws s3 ls $DEST > /dev/null ; then
        echo -e "${YELLOW}${DEST} already exists, will not update${RESET}"
    else
        echo -e "${GREEN}Uploading ${FILE} to ${DEST}...${RESET}"
        aws s3 cp $FILE $DEST
        UPLOADED=true
    fi
done

if [[ "$VERSION" =~ .*(alpha|beta|pre|rc).* ]]; then
    echo -e "${YELLOW}Pre-release version found, will not update latest_version.${RESET}"
elif [ "$UPLOADED" = true ]; then
    echo -e "${GREEN}Uploaded at least one file, updating latest_version...${RESET}"
    aws s3 cp "VERSION" "${BUCKET}/latest_version"
else
    echo -e "${YELLOW}Didn't upload anything, will not update latest_version${RESET}"
fi
