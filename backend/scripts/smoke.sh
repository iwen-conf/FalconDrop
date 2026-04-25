#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/deployments/docker-compose.yml"
FIXTURE_FILE="${ROOT_DIR}/scripts/smoke-upload.txt"
COOKIE_FILE=""

on_error() {
  local exit_code=$?
  echo "[smoke] failed (exit=${exit_code}), dumping compose status"
  docker compose -f "${COMPOSE_FILE}" ps || true
  docker compose -f "${COMPOSE_FILE}" logs --tail=120 app || true
  docker compose -f "${COMPOSE_FILE}" logs --tail=80 postgres || true
  exit "${exit_code}"
}
trap on_error ERR

cleanup() {
  rm -f "${COOKIE_FILE:-}" "${FIXTURE_FILE}"
}
trap cleanup EXIT

if ! command -v docker >/dev/null 2>&1; then
  echo "[smoke] docker not found"
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "[smoke] curl not found"
  exit 1
fi

echo "[smoke] compose file: ${COMPOSE_FILE}"
echo "[smoke] start services"
docker compose -f "${COMPOSE_FILE}" up -d --build

echo "[smoke] wait app ready"
ready=0
for i in {1..60}; do
  if curl -fsS "http://127.0.0.1:8080/readyz" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
if [[ "${ready}" -ne 1 ]]; then
  echo "[smoke] app not ready after 120s"
  exit 1
fi

echo "[smoke] login"
COOKIE_FILE="$(mktemp)"

curl -fsS -c "${COOKIE_FILE}" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${DEFAULT_SYSTEM_USERNAME:-admin}\",\"password\":\"${DEFAULT_SYSTEM_PASSWORD:-change-me}\"}" \
  "http://127.0.0.1:8080/api/auth/login" >/dev/null

echo "[smoke] auth/me"
curl -fsS -b "${COOKIE_FILE}" "http://127.0.0.1:8080/api/auth/me" >/dev/null

echo "[smoke] ftp/start"
curl -fsS -b "${COOKIE_FILE}" -X POST "http://127.0.0.1:8080/api/ftp/start" >/dev/null

echo "[smoke] ftp upload"
echo "falcondrop smoke $(date -u +%Y-%m-%dT%H:%M:%SZ)" > "${FIXTURE_FILE}"
curl -fsS --ftp-create-dirs -T "${FIXTURE_FILE}" \
  "ftp://${DEFAULT_FTP_USERNAME:-camera}:${DEFAULT_FTP_PASSWORD:-change-me}@127.0.0.1:2121/smoke/smoke-upload.txt" >/dev/null

echo "[smoke] list photos"
curl -fsS -b "${COOKIE_FILE}" "http://127.0.0.1:8080/api/photos" >/dev/null

echo "[smoke] list assets"
ASSETS_JSON="$(curl -fsS -b "${COOKIE_FILE}" "http://127.0.0.1:8080/api/assets?limit=20")"
echo "${ASSETS_JSON}" | grep -q "smoke-upload.txt" || {
  echo "[smoke] uploaded file not found in assets"
  exit 1
}

echo "[smoke] ftp/stop"
curl -fsS -b "${COOKIE_FILE}" -X POST "http://127.0.0.1:8080/api/ftp/stop" >/dev/null

echo "[smoke] done"
