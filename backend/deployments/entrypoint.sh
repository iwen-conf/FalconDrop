#!/bin/sh
set -eu

mkdir -p "${STORAGE_ROOT:-/data/falcondrop/uploads}" "${TMP_ROOT:-/data/falcondrop/tmp}"
chown -R app:app "${STORAGE_ROOT:-/data/falcondrop/uploads}" "${TMP_ROOT:-/data/falcondrop/tmp}"

exec su-exec app:app /app/falcondrop-api
