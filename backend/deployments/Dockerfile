FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/falcondrop-api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache su-exec \
  && addgroup -S app \
  && adduser -S app -G app \
  && mkdir -p /data/falcondrop/uploads /data/falcondrop/tmp

WORKDIR /app
COPY --from=builder /out/falcondrop-api /app/falcondrop-api
COPY deployments/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh \
  && chown -R app:app /app /data

USER root
EXPOSE 8080 2121 30000-30009

ENTRYPOINT ["/entrypoint.sh"]
