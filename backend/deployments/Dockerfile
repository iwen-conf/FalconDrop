FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/falcondrop-api ./cmd/api

FROM alpine:3.21

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/falcondrop-api /app/falcondrop-api

USER app
EXPOSE 8080 2121 30000-30009

ENTRYPOINT ["/app/falcondrop-api"]
