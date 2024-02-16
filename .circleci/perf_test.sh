#!/usr/bin/env bash

set -eu

DIR="${1:-/tmp/please}"

echo "Extracting Please..."
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/linux_amd64/please_*.tar.gz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

BUCKET="gs://please.build/performance"

echo "Generating test file tree..."
/tmp/workspace/gen_parse_tree.pex --plz plz --noprogress --size 300000

echo "Running parse performance test..."
/tmp/workspace/parse_perf_test.pex --plz plz --revision "$CIRCLE_SHA1"

echo "Uploading results..."
# Auth against gcp for cli and
echo $GCLOUD_SERVICE_KEY > $GOOGLE_APPLICATION_CREDENTIALS
echo $GCLOUD_SERVICE_KEY | gcloud auth activate-service-account --key-file=-

gsutil cp plz.prof "${BUCKET}/${CIRCLE_SHA1}.prof"
gsutil cp results.json "${BUCKET}/${CIRCLE_SHA1}.json"
if gsutil ls "${BUCKET}/all_results.jsonl"; then
    gsutil cp "${BUCKET}/all_results.jsonl" all_results.jsonl
    cat all_results.jsonl results.json | tail -n 100 > updated_results.jsonl
    gsutil cp updated_results.jsonl "${BUCKET}/all_results.jsonl"
else
    gsutil cp results.json "${BUCKET}/all_results.jsonl"
fi

rm -rf tree

echo "Running benchmarks..."
plz build -i benchmark -p -v notice -o "buildconfig.benchmark-revision:${CIRCLE_SHA1}"

for RESULT in plz-out/benchmarks/*.json; do
  BENCHMARK_NAME=${RESULT%.json}
  BENCHMARK_NAME=${BENCHMARK_NAME#plz-out/benchmarks/}

  echo "Uploading ${BENCHMARK_NAME} results..."

  gsutil cp "$RESULT" "${BUCKET}/${BENCHMARK_NAME}_${CIRCLE_SHA1}.json"

  ALL_RESULTS="${BENCHMARK_NAME}_all_results.jsonl"
  if gsutil ls "${BUCKET}/${ALL_RESULTS}"; then
      gsutil cp "${BUCKET}/${ALL_RESULTS}" "${ALL_RESULTS}"
      cat "${ALL_RESULTS}" "${RESULT}" | tail -n 100 > "updated_${ALL_RESULTS}"
      gsutil cp "updated_${ALL_RESULTS}" "${BUCKET}/${ALL_RESULTS}"
  else
      gsutil cp "${RESULT}" "${BUCKET}/${ALL_RESULTS}"
  fi
done

echo "Done!"
