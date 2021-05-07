#!/usr/bin/env bash

set -eu

VER="`cat VERSION`"
PLZ="/tmp/workspace/linux_amd64/please_${VER}"
BUCKET="s3://please-releases/performance"

echo "Generating test file tree"
/tmp/workspace/gen_parse_tree.pex --plz "$PLZ" --noprogress --size 500000
echo "Running parse performance test"
/tmp/workspace/parse_perf_test.pex --plz "$PLZ" --revision "$CIRCLE_SHA1"
echo "Uploading results..."
aws s3 cp plz.prof "${BUCKET}/${CIRCLE_SHA1}.prof"
aws s3 cp results.json "${BUCKET}/${CIRCLE_SHA1}.json"
if aws s3 ls "${BUCKET}/all_results.jsonl"; then
    aws s3 cp "${BUCKET}/all_results.jsonl" all_results.jsonl
    cat all_results.jsonl results.json | tail -n 100 > updated_results.jsonl
    aws s3 cp updated_results.jsonl "${BUCKET}/all_results.jsonl"
else
    aws s3 cp results.json "${BUCKET}/all_results.jsonl"
fi
aws s3 cp tools/performance/report.html "${BUCKET}/report.html"
echo "Done!"
