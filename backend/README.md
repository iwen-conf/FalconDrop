# FalconDrop Backend

## Run

```bash
cp .env.example .env
go run ./cmd/api
```

## Checks

```bash
go test ./...
go vet ./...
docker compose -f deployments/docker-compose.yml config
```

## Integration Test

Requires a running PostgreSQL and writable storage dirs.

```bash
go test -tags=integration ./internal/api -run TestIntegrationAuthLoginAndMe
```

Enable full FTP + WebSocket + delete consistency integration:

```bash
RUN_FTP_INTEGRATION=1 go test -tags=integration ./internal/api -run TestIntegrationFTPUploadAssetAndDeletePhotoWithWebSocket
```

## Smoke Script

Requires Docker daemon and `curl`.

```bash
./scripts/smoke.sh
```

When Docker is unavailable in the current host session, this script will fail at startup.

## FTP Notes

- FTP control port: `2121`
- Passive ports: `30000-30009`
- In Docker/NAT, set `FTP_PUBLIC_HOST` to the reachable host/IP.
