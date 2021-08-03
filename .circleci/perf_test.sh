#!/usr/bin/env bash

set -eu

DIR="${1:-/tmp/please}"

echo "Extracting Please..."
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/linux_amd64/please_*.tar.gz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

BUCKET="s3://please-docs/performance"

echo "Generating test file tree..."
/tmp/workspace/gen_parse_tree.pex --plz plz --noprogress --size 300000

echo "Running parse performance test..."
/tmp/workspace/parse_perf_test.pex --plz plz --revision "$CIRCLE_SHA1"

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

rm -rf tree

echo "Running benchmarks..."
plz build -i benchmark -p -v notice -o "buildconfig.benchmark-revision:${CIRCLE_SHA1}"

for RESULT in plz-out/benchmarks/*.json; do
  BENCHMARK_NAME=${RESULT%.json}
  BENCHMARK_NAME=${BENCHMARK_NAME#plz-out/benchmarks/}

  echo "Uploading ${BENCHMARK_NAME} results..."

  aws s3 cp "$RESULT" "${BUCKET}/${BENCHMARK_NAME}_${CIRCLE_SHA1}.json"

  ALL_RESULTS="${BENCHMARK_NAME}_all_results.jsonl"
  if aws s3 ls "${BUCKET}/${ALL_RESULTS}"; then
      aws s3 cp "${BUCKET}/${ALL_RESULTS}" "${ALL_RESULTS}"
      cat "${ALL_RESULTS}" "${RESULT}" | tail -n 100 > "updated_${ALL_RESULTS}"
      aws s3 cp "updated_${ALL_RESULTS}" "${BUCKET}/${ALL_RESULTS}"
  else
      aws s3 cp "${RESULT}" "${BUCKET}/${ALL_RESULTS}"
  fi
done

echo "Done!"
