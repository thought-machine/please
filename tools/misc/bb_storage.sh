#!/bin/sh

mkdir -p plz-out/storage-cas plz-out/storage-ac

exec docker run \
      -p 8980:8980 \
      -v $(pwd)/tools/misc:/config \
      -v $(pwd)/plz-out/storage-cas:/storage-cas \
      -v $(pwd)/plz-out/storage-ac:/storage-ac \
      buildbarn/bb-storage:20190617T155413Z-3c42fa4 \
      /config/bb_storage.json
