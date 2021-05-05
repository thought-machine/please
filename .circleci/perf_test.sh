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
aws s3 cp s3://please-releases/performance/all_results.jsonl all_results.jsonl
cat all_results.jsonl results.json | tail -n 100 > updated_results.jsonl
aws s3 cp updated_results.jsonl s3://please-releases/performance/all_results.jsonl
echo "Done!"
