#!/usr/bin/env bash

set -eu

PLZ="/tmp/workspace/linux_amd64/please"
REV="`git rev-parse HEAD`"

echo "Generating test file tree"
/tmp/workspace/gen_parse_tree.pex -- --plz "$PLZ"
echo "Running parse performance test"
/tmp/workspace/parse_perf_test.pex -- --plz "$PLZ"
echo "Uploading results..."
aws s3 cp plz.prof "s3://please-releases/performance/${REVISION}.prof"
aws s3 cp results.json "s3://please-releases/performance/${REVISION}.json"
aws s3 cp s3://please-releases/performance/all_results.json all_results.json
cat all_results.json results.json | tail -n 100 > updated_results.json
aws s3 cp updated_results.json s3://please-releases/performance/all_results.json
echo "Done!"
