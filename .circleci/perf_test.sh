#!/usr/bin/env bash

set -eu

PLZ="/tmp/workspace/linux_amd64/please"
REV="`git rev-parse HEAD`"

echo "Generating test file tree"
"$PLZ" run //tools/misc:gen_parse_tree -- --plz "$PLZ"
echo "Running parse performance test"
"$PLZ" run //tools/performance:parse_perf_test -- --plz "$PLZ"
echo "Uploading results..."
aws s3 cp plz.prof "s3://please-releases/performance/${REVISION}.prof"
aws s3 cp results.json "s3://please-releases/performance/${REVISION}.json"
aws s3 cp s3://please-releases/performance/all_results.json all_results.json
cat all_results.json results.json > updated_results.json
aws s3 cp updated_results.json s3://please-releases/performance/all_results.json
echo "Done!"
