#!/usr/bin/env bash

set -euo pipefail

API_URL="${API_URL:-http://localhost:8080/render}"
OUTPUT_FILE="${OUTPUT_FILE:-main.tf}"

if [[ -n "${1:-}" ]]; then
  if [[ ! -f "$1" ]]; then
    echo "Error: File '$1' not found"
    exit 1
  fi
  PAYLOAD=$(cat "$1")
else
  PAYLOAD='{
  "payload": {
    "properties": {
      "aws-region": "us-west-1",
      "acl": "private",
      "bucket-name": "triplaaaa-bucket"
    }
  }
}'
fi

echo "Sending request to ${API_URL}..."

BODY_FILE=$(mktemp)
STATUS_FILE=$(mktemp)

curl -s \
  -o "${BODY_FILE}" \
  -w "%{http_code}" \
  -X POST "${API_URL}" \
  -H "Content-Type: application/json" \
  -d "${PAYLOAD}" > "${STATUS_FILE}"

HTTP_STATUS=$(cat "${STATUS_FILE}")

if [[ "${HTTP_STATUS}" == "200" ]]; then
  mv "${BODY_FILE}" "${OUTPUT_FILE}"
  echo "Terraform file generated: ${OUTPUT_FILE}"
else
  echo "Request failed: ${HTTP_STATUS}"
  echo "Response:"
  cat "${BODY_FILE}"
  rm -f "${BODY_FILE}"
  exit 1
fi

rm -f "${STATUS_FILE}"
